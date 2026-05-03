// Package card implements the card data model for focus.
//
// A card on disk is a directory named "<padded-id>-<slug>/" containing
// at minimum an INDEX.md file with YAML frontmatter and a markdown
// body. This package parses, validates, and re-marshals cards while
// preserving unknown frontmatter fields per the schema rule in
// designs/focus-v2.md ("Unknown fields are preserved on read/write").
package card

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// SchemaVersion is the only frontmatter schema this binary speaks. The
// CLI refuses to operate on cards with a missing or unknown
// schema_version (designs/focus-v2.md §"Schema versioning").
const SchemaVersion = 2

// Type is the card's type. v2 has two: regular cards and epics.
type Type string

const (
	TypeCard Type = "card"
	TypeEpic Type = "epic"
)

// Status is the card's lifecycle position. Four values, no more
// (designs/focus-v2.md §"Statuses").
type Status string

const (
	StatusActive   Status = "active"
	StatusBacklog  Status = "backlog"
	StatusDone     Status = "done"
	StatusArchived Status = "archived"
)

// Priority is the card's priority. p0 is highest.
type Priority string

const (
	PriorityP0 Priority = "p0"
	PriorityP1 Priority = "p1"
	PriorityP2 Priority = "p2"
	PriorityP3 Priority = "p3"
)

// Card is the parsed, in-memory representation of a focus card.
//
// The strongly-typed fields cover the required + optional set defined
// in the schema. Anything else found in the frontmatter is held in
// Extra and round-tripped on Marshal so users can extend frontmatter
// without forking the binary.
type Card struct {
	// Required fields. The CLI rejects cards on disk that are missing
	// any of these.
	SchemaVersion int       `yaml:"schema_version"`
	ID            int       `yaml:"id"`
	UUID          string    `yaml:"uuid"`
	Title         string    `yaml:"title"`
	Type          Type      `yaml:"type"`
	Status        Status    `yaml:"status"`
	Priority      Priority  `yaml:"priority"`
	Project       string    `yaml:"project"`
	Created       time.Time `yaml:"created"`

	// Optional but recognized fields.
	Epic        *int     `yaml:"epic,omitempty"`
	DependsOn   []int    `yaml:"depends-on,omitempty"`
	Contract    []string `yaml:"contract,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
	Owner       string   `yaml:"owner,omitempty"`
	Description string   `yaml:"description,omitempty"`
	Area        string   `yaml:"area,omitempty"`

	// Extra holds any frontmatter keys this binary doesn't recognize.
	// Values come from yaml.v3 decode and are re-encoded on Marshal so
	// downstream extensions survive a CLI write round-trip.
	Extra map[string]any `yaml:"-"`

	// Body is the markdown content of the card after the closing "---"
	// of the frontmatter block. Stored verbatim including the trailing
	// newline (or lack thereof) so round-trips are byte-clean as long
	// as nothing in the frontmatter changes.
	Body string `yaml:"-"`
}

// knownFields is the set of frontmatter keys handled by typed Card
// fields. Anything not in this set goes to Extra.
var knownFields = map[string]bool{
	"schema_version": true,
	"id":             true,
	"uuid":           true,
	"title":          true,
	"type":           true,
	"status":         true,
	"priority":       true,
	"project":        true,
	"created":        true,
	"epic":           true,
	"depends-on":     true,
	"contract":       true,
	"tags":           true,
	"owner":          true,
	"description":    true,
	"area":           true,
}

// Parse reads an INDEX.md file's bytes and returns a Card.
//
// The frontmatter delimiter is the canonical "---\n...---\n" form. We
// don't accept TOML or JSON frontmatter — focus is YAML-only, and
// adrg/frontmatter's auto-detection is therefore overkill. We do the
// split ourselves to keep unknown-field handling under our control.
func Parse(data []byte) (*Card, error) {
	fm, body, err := splitFrontmatter(data)
	if err != nil {
		return nil, err
	}

	// Decode twice: once into a map (to find unknown fields) and once
	// into the typed Card. Going through a map first means we don't
	// need yaml.v3's custom UnmarshalYAML hooks to populate Extra,
	// which keeps the typed struct boring and easy to reason about.
	var raw map[string]any
	if err := yaml.Unmarshal(fm, &raw); err != nil {
		return nil, fmt.Errorf("frontmatter yaml: %w", err)
	}

	c := &Card{Body: string(body)}
	if err := yaml.Unmarshal(fm, c); err != nil {
		return nil, fmt.Errorf("frontmatter yaml typed decode: %w", err)
	}

	// Anything in raw but not in our known field set is preserved.
	for k, v := range raw {
		if !knownFields[k] {
			if c.Extra == nil {
				c.Extra = make(map[string]any)
			}
			c.Extra[k] = v
		}
	}

	return c, nil
}

// Marshal serializes a Card back into INDEX.md bytes, preserving
// unknown fields from c.Extra. Field order in the frontmatter is
// stable: required fields first (in the order they appear in the
// design doc), then optional recognized fields, then Extra in
// alphabetical order. Stable order means git diffs stay readable when
// CLI writes touch a card.
func Marshal(c *Card) ([]byte, error) {
	// Build an ordered yaml.Node so we control the key order. yaml.v3's
	// Node API is the only way to get deterministic output without
	// custom MarshalYAML methods on Card.
	root := &yaml.Node{Kind: yaml.MappingNode}

	addStr := func(key, val string) {
		if val == "" {
			return
		}
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: key},
			&yaml.Node{Kind: yaml.ScalarNode, Value: val},
		)
	}
	addInt := func(key string, val int) {
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: key},
			&yaml.Node{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%d", val)},
		)
	}
	addAny := func(key string, val any) error {
		var n yaml.Node
		if err := n.Encode(val); err != nil {
			return err
		}
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: key},
			&n,
		)
		return nil
	}

	addInt("schema_version", c.SchemaVersion)
	addInt("id", c.ID)
	addStr("uuid", c.UUID)
	addStr("title", c.Title)
	addStr("type", string(c.Type))
	addStr("status", string(c.Status))
	addStr("priority", string(c.Priority))
	addStr("project", c.Project)
	if !c.Created.IsZero() {
		addStr("created", c.Created.Format("2006-01-02"))
	}

	if c.Epic != nil {
		addInt("epic", *c.Epic)
	}
	if len(c.DependsOn) > 0 {
		if err := addAny("depends-on", c.DependsOn); err != nil {
			return nil, err
		}
	}
	if len(c.Contract) > 0 {
		if err := addAny("contract", c.Contract); err != nil {
			return nil, err
		}
	}
	if len(c.Tags) > 0 {
		if err := addAny("tags", c.Tags); err != nil {
			return nil, err
		}
	}
	addStr("owner", c.Owner)
	addStr("description", c.Description)
	addStr("area", c.Area)

	// Extra in alphabetical order for diff stability.
	if len(c.Extra) > 0 {
		keys := make([]string, 0, len(c.Extra))
		for k := range c.Extra {
			keys = append(keys, k)
		}
		// avoid pulling in sort just for this; manual insertion sort is
		// fine at the typical Extra size (single digits).
		for i := 1; i < len(keys); i++ {
			for j := i; j > 0 && keys[j-1] > keys[j]; j-- {
				keys[j-1], keys[j] = keys[j], keys[j-1]
			}
		}
		for _, k := range keys {
			if err := addAny(k, c.Extra[k]); err != nil {
				return nil, err
			}
		}
	}

	var fm bytes.Buffer
	enc := yaml.NewEncoder(&fm)
	enc.SetIndent(2)
	if err := enc.Encode(root); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}

	var out bytes.Buffer
	out.WriteString("---\n")
	out.Write(fm.Bytes())
	out.WriteString("---\n")
	out.WriteString(c.Body)
	return out.Bytes(), nil
}

// splitFrontmatter splits a card file's bytes into the frontmatter
// YAML and the body. The expected shape is:
//
//	---
//	<yaml>
//	---
//	<body>
//
// A leading BOM is tolerated; trailing whitespace before the second
// "---" is allowed. If the file does not start with a "---" line we
// return an error: focus cards are always frontmatter-prefixed.
func splitFrontmatter(data []byte) (fm, body []byte, err error) {
	// Strip optional UTF-8 BOM. Editors love to add it.
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})

	// Accept "---\n" or "---\r\n" as the opening line.
	open := bytes.IndexByte(data, '\n')
	if open == -1 {
		return nil, nil, fmt.Errorf("card has no frontmatter delimiter")
	}
	first := bytes.TrimRight(data[:open], "\r")
	if string(first) != "---" {
		return nil, nil, fmt.Errorf("card does not start with frontmatter delimiter %q, got %q", "---", string(first))
	}
	rest := data[open+1:]

	// Find the closing delimiter line. Walk line by line so we can
	// match exactly "---" without false-positives on body content.
	closeIdx := -1
	for i := 0; i < len(rest); {
		nl := bytes.IndexByte(rest[i:], '\n')
		var line []byte
		if nl == -1 {
			line = rest[i:]
		} else {
			line = rest[i : i+nl]
		}
		if string(bytes.TrimRight(line, "\r")) == "---" {
			closeIdx = i
			break
		}
		if nl == -1 {
			break
		}
		i += nl + 1
	}
	if closeIdx == -1 {
		return nil, nil, fmt.Errorf("card frontmatter is not closed by --- delimiter")
	}
	fm = rest[:closeIdx]
	// body starts after the "---" line + its newline. Skip one or two
	// chars (\n or \r\n).
	rest2 := rest[closeIdx:]
	nl := bytes.IndexByte(rest2, '\n')
	if nl == -1 {
		body = nil
	} else {
		body = rest2[nl+1:]
	}
	return fm, body, nil
}

// Validate returns an error if any required field is missing, has the
// wrong shape, or carries an unknown schema_version. This is the gate
// run on every card load — see designs/focus-v2.md §"Required vs
// optional fields".
func (c *Card) Validate() error {
	if c.SchemaVersion != SchemaVersion {
		return fmt.Errorf("schema_version %d unsupported (this binary speaks %d)", c.SchemaVersion, SchemaVersion)
	}
	if c.ID <= 0 {
		return fmt.Errorf("id must be a positive integer, got %d", c.ID)
	}
	if c.UUID == "" {
		return fmt.Errorf("uuid is required")
	}
	if strings.TrimSpace(c.Title) == "" {
		return fmt.Errorf("title is required")
	}
	switch c.Type {
	case TypeCard, TypeEpic:
	default:
		return fmt.Errorf("type %q invalid (must be card or epic)", c.Type)
	}
	switch c.Status {
	case StatusActive, StatusBacklog, StatusDone, StatusArchived:
	default:
		return fmt.Errorf("status %q invalid", c.Status)
	}
	switch c.Priority {
	case PriorityP0, PriorityP1, PriorityP2, PriorityP3:
	default:
		return fmt.Errorf("priority %q invalid (must be p0|p1|p2|p3)", c.Priority)
	}
	if strings.TrimSpace(c.Project) == "" {
		return fmt.Errorf("project is required")
	}
	if c.Created.IsZero() {
		return fmt.Errorf("created date is required")
	}
	return nil
}

// PaddedID returns the 4-digit zero-padded form of the card id used in
// the directory name (e.g. 142 → "0142"). Lex-sort over directory
// names matches numeric sort up to id 9999, which is the upper bound
// of v2's design.
func PaddedID(id int) string {
	return fmt.Sprintf("%04d", id)
}

// DirName returns the canonical directory name for a card,
// "<padded-id>-<slug>". slug is expected to be already-normalized.
func DirName(id int, slug string) string {
	return PaddedID(id) + "-" + slug
}
