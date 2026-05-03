package card

import (
	"strings"
	"testing"
	"time"
)

const sampleCard = `---
schema_version: 2
id: 142
uuid: 7f3a9b2c-9e1d-4f8a-b5e1-6e2d8f1a3c4b
title: Ship the feature
type: card
status: backlog
priority: p2
project: api
created: 2026-05-04
epic: 5
contract:
  - Tests pass
  - Code reviewed
tags:
  - backend
  - mcp
owner: ash
custom_field: hello
nested_extra:
  alpha: 1
  beta: two
---
## Summary

Free-form markdown body.
`

func TestParseHappyPath(t *testing.T) {
	c, err := Parse([]byte(sampleCard))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if c.ID != 142 {
		t.Errorf("ID = %d, want 142", c.ID)
	}
	if c.UUID != "7f3a9b2c-9e1d-4f8a-b5e1-6e2d8f1a3c4b" {
		t.Errorf("UUID = %q", c.UUID)
	}
	if c.Title != "Ship the feature" {
		t.Errorf("Title = %q", c.Title)
	}
	if c.Type != TypeCard {
		t.Errorf("Type = %q", c.Type)
	}
	if c.Status != StatusBacklog {
		t.Errorf("Status = %q", c.Status)
	}
	if c.Priority != PriorityP2 {
		t.Errorf("Priority = %q", c.Priority)
	}
	if c.Project != "api" {
		t.Errorf("Project = %q", c.Project)
	}
	wantCreated := time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC)
	if !c.Created.Equal(wantCreated) {
		t.Errorf("Created = %v, want %v", c.Created, wantCreated)
	}
	if c.Epic == nil || *c.Epic != 5 {
		t.Errorf("Epic = %v", c.Epic)
	}
	if len(c.Contract) != 2 {
		t.Errorf("Contract len = %d, want 2", len(c.Contract))
	}
	if len(c.Tags) != 2 {
		t.Errorf("Tags len = %d", len(c.Tags))
	}
	if c.Owner != "ash" {
		t.Errorf("Owner = %q", c.Owner)
	}
	if c.Extra["custom_field"] != "hello" {
		t.Errorf("Extra[custom_field] = %v", c.Extra["custom_field"])
	}
	if _, ok := c.Extra["nested_extra"]; !ok {
		t.Error("Extra[nested_extra] missing")
	}
	if !strings.Contains(c.Body, "Free-form markdown body.") {
		t.Errorf("Body lost markdown content: %q", c.Body)
	}
}

func TestRoundTripPreservesUnknownFields(t *testing.T) {
	c, err := Parse([]byte(sampleCard))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	out, err := Marshal(c)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	c2, err := Parse(out)
	if err != nil {
		t.Fatalf("Parse(Marshal(c)): %v", err)
	}
	if c2.Extra["custom_field"] != "hello" {
		t.Errorf("custom_field lost: %v", c2.Extra["custom_field"])
	}
	if _, ok := c2.Extra["nested_extra"]; !ok {
		t.Error("nested_extra lost on round-trip")
	}
	if c2.Title != c.Title || c2.ID != c.ID || c2.UUID != c.UUID {
		t.Error("identity fields drifted on round-trip")
	}
}

func TestMarshalIncludesBody(t *testing.T) {
	c, err := Parse([]byte(sampleCard))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	out, err := Marshal(c)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(out), "Free-form markdown body.") {
		t.Errorf("body missing from marshal output:\n%s", out)
	}
	if !strings.HasPrefix(string(out), "---\n") {
		t.Errorf("marshal output missing leading ---: %q", string(out)[:10])
	}
}

func TestValidateRejectsBadSchemaVersion(t *testing.T) {
	c := goodCard()
	c.SchemaVersion = 1
	if err := c.Validate(); err == nil {
		t.Error("Validate should reject schema_version != 2")
	}
}

func TestValidateRejectsBadPriority(t *testing.T) {
	c := goodCard()
	c.Priority = "p9"
	if err := c.Validate(); err == nil {
		t.Error("Validate should reject p9")
	}
}

func TestValidateRejectsBadStatus(t *testing.T) {
	c := goodCard()
	c.Status = "killed"
	if err := c.Validate(); err == nil {
		t.Error("Validate should reject killed status (v1 holdover)")
	}
}

func TestValidateRejectsBadType(t *testing.T) {
	c := goodCard()
	c.Type = "milestone"
	if err := c.Validate(); err == nil {
		t.Error("Validate should reject milestone type (v1 holdover)")
	}
}

func TestValidateRequiresAllFields(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*Card)
	}{
		{"id", func(c *Card) { c.ID = 0 }},
		{"uuid", func(c *Card) { c.UUID = "" }},
		{"title", func(c *Card) { c.Title = "" }},
		{"project", func(c *Card) { c.Project = "" }},
		{"created", func(c *Card) { c.Created = time.Time{} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := goodCard()
			tc.mut(c)
			if err := c.Validate(); err == nil {
				t.Errorf("Validate accepted card missing %s", tc.name)
			}
		})
	}
}

func TestParseRejectsMissingFrontmatter(t *testing.T) {
	if _, err := Parse([]byte("just a body, no fm\n")); err == nil {
		t.Error("Parse should reject body without frontmatter")
	}
}

func TestParseRejectsUnclosedFrontmatter(t *testing.T) {
	if _, err := Parse([]byte("---\nid: 1\nbody without close\n")); err == nil {
		t.Error("Parse should reject unclosed frontmatter")
	}
}

func TestParseHandlesCRLF(t *testing.T) {
	// Editors on macOS sometimes save CRLF. Round-trip should tolerate.
	in := strings.ReplaceAll(sampleCard, "\n", "\r\n")
	if _, err := Parse([]byte(in)); err != nil {
		t.Errorf("Parse rejected CRLF input: %v", err)
	}
}

func TestParseHandlesUTF8BOM(t *testing.T) {
	in := []byte("\xEF\xBB\xBF" + sampleCard)
	if _, err := Parse(in); err != nil {
		t.Errorf("Parse rejected UTF-8 BOM prefix: %v", err)
	}
}

func TestPaddedID(t *testing.T) {
	cases := map[int]string{
		1:    "0001",
		42:   "0042",
		142:  "0142",
		9999: "9999",
	}
	for n, want := range cases {
		if got := PaddedID(n); got != want {
			t.Errorf("PaddedID(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestDirName(t *testing.T) {
	if got := DirName(142, "ship-the-feature"); got != "0142-ship-the-feature" {
		t.Errorf("DirName = %q", got)
	}
}

func goodCard() *Card {
	return &Card{
		SchemaVersion: SchemaVersion,
		ID:            1,
		UUID:          "abc",
		Title:         "Test card",
		Type:          TypeCard,
		Status:        StatusBacklog,
		Priority:      PriorityP2,
		Project:       "test",
		Created:       time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC),
	}
}
