package card

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// MarshalUpdate returns INDEX.md bytes for c using c.Raw as the
// byte-preserving baseline. Only the frontmatter keys whose values
// changed since c was Parse'd are rewritten; unchanged lines pass
// through verbatim (including blank lines, indentation, comments on
// their own line, and the original flow style of unchanged keys).
// The body region after the closing "---" passes through verbatim.
//
// Caveat: rewriting a scalar value drops a trailing inline comment
// on that same line, since the YAML decoder doesn't surface it.
// Comments on their own lines are unaffected.
//
// Falls back to Marshal if c.Raw is empty (a card built from scratch
// has nothing to preserve).
func MarshalUpdate(c *Card) ([]byte, error) {
	if len(c.Raw) == 0 {
		return Marshal(c)
	}
	origFM, _, err := splitFrontmatter(c.Raw)
	if err != nil {
		return Marshal(c)
	}

	orig, err := Parse(c.Raw)
	if err != nil {
		return Marshal(c)
	}

	updates, inserts, err := diffFrontmatter(orig, c)
	if err != nil {
		return nil, err
	}

	newFM, err := applyFrontmatterEdits(origFM, updates, inserts)
	if err != nil {
		return nil, err
	}

	eol := detectEOL(c.Raw)

	var out bytes.Buffer
	out.WriteString("---" + eol)
	out.Write(newFM)
	if len(newFM) > 0 && newFM[len(newFM)-1] != '\n' {
		out.WriteString(eol)
	}
	out.WriteString("---" + eol)
	out.WriteString(c.Body)
	return out.Bytes(), nil
}

// detectEOL inspects the original card bytes and returns "\r\n" when
// the file uses CRLF line endings, "\n" otherwise. Used so
// MarshalUpdate emits frontmatter delimiters in the file's native
// EOL style instead of forcing LF on a CRLF file.
func detectEOL(raw []byte) string {
	for i := 0; i < len(raw); i++ {
		if raw[i] == '\n' {
			if i > 0 && raw[i-1] == '\r' {
				return "\r\n"
			}
			return "\n"
		}
	}
	return "\n"
}

func diffFrontmatter(orig, next *Card) (updates map[string]string, inserts []keyValue, err error) {
	updates = map[string]string{}

	if orig.Status != next.Status {
		updates["status"] = string(next.Status)
	}
	if orig.Priority != next.Priority {
		updates["priority"] = string(next.Priority)
	}
	if orig.Title != next.Title {
		updates["title"] = next.Title
	}
	if orig.Project != next.Project {
		updates["project"] = next.Project
	}
	if orig.Type != next.Type {
		updates["type"] = string(next.Type)
	}
	if orig.Owner != next.Owner {
		updates["owner"] = next.Owner
	}
	if orig.Description != next.Description {
		updates["description"] = next.Description
	}
	if orig.Area != next.Area {
		updates["area"] = next.Area
	}
	if orig.UUID != next.UUID {
		updates["uuid"] = next.UUID
	}
	if orig.ID != next.ID {
		updates["id"] = fmt.Sprintf("%d", next.ID)
	}
	if orig.SchemaVersion != next.SchemaVersion {
		updates["schema_version"] = fmt.Sprintf("%d", next.SchemaVersion)
	}

	if !equalIntPtr(orig.Epic, next.Epic) {
		if next.Epic == nil {
			updates["epic"] = ""
		} else {
			updates["epic"] = fmt.Sprintf("%d", *next.Epic)
		}
	}

	if err := requireUnchanged("contract", orig.Contract, next.Contract); err != nil {
		return nil, nil, err
	}
	if err := requireUnchanged("tags", orig.Tags, next.Tags); err != nil {
		return nil, nil, err
	}
	if err := requireUnchangedInts("depends-on", orig.DependsOn, next.DependsOn); err != nil {
		return nil, nil, err
	}

	if !equalDate(orig.Created, next.Created) {
		updates["created"] = next.Created.Format("2006-01-02")
	}

	for k, v := range updates {
		if v == "" && k == "epic" {
			continue
		}
		if !hasTopLevelKey(orig, k) {
			inserts = append(inserts, keyValue{key: k, value: v})
			delete(updates, k)
		}
	}

	return updates, inserts, nil
}

type keyValue struct {
	key   string
	value string
}

func hasTopLevelKey(c *Card, key string) bool {
	if len(c.Raw) == 0 {
		return false
	}
	fm, _, err := splitFrontmatter(c.Raw)
	if err != nil {
		return false
	}
	var node yaml.Node
	if err := yaml.Unmarshal(fm, &node); err != nil {
		return false
	}
	if node.Kind != yaml.DocumentNode || len(node.Content) == 0 {
		return false
	}
	mapping := node.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return false
	}
	for i := 0; i < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return true
		}
	}
	return false
}

func requireUnchanged(name string, a, b []string) error {
	if len(a) != len(b) {
		return fmt.Errorf("byte-preserving save cannot rewrite list field %q (length %d → %d); reload card and retry", name, len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			return fmt.Errorf("byte-preserving save cannot rewrite list field %q (item %d changed)", name, i)
		}
	}
	return nil
}

func requireUnchangedInts(name string, a, b []int) error {
	if len(a) != len(b) {
		return fmt.Errorf("byte-preserving save cannot rewrite list field %q", name)
	}
	for i := range a {
		if a[i] != b[i] {
			return fmt.Errorf("byte-preserving save cannot rewrite list field %q", name)
		}
	}
	return nil
}

func equalIntPtr(a, b *int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func equalDate(a, b interface{ Format(string) string }) bool {
	return a.Format("2006-01-02") == b.Format("2006-01-02")
}

// applyFrontmatterEdits walks the original frontmatter line-by-line
// and rewrites the scalar value of each top-level key in updates.
// Lines belonging to nested mappings, list items, or block scalars
// are passed through unchanged so flow-style and comments stay put.
//
// inserts are appended at the end of the frontmatter (preserves
// original key order for everything else).
func applyFrontmatterEdits(fm []byte, updates map[string]string, inserts []keyValue) ([]byte, error) {
	if len(updates) == 0 && len(inserts) == 0 {
		return fm, nil
	}

	lines := splitLinesKeepEnding(fm)
	out := make([][]byte, 0, len(lines))

	inBlockScalar := false
	for i, line := range lines {
		text := stripLineEnding(line)
		if inBlockScalar {
			if isTopLevelKeyLine(text) {
				inBlockScalar = false
			} else {
				out = append(out, line)
				continue
			}
		}

		key, _, hasValue, isTL := parseTopLevelScalarLine(text)
		if !isTL {
			out = append(out, line)
			continue
		}

		if !hasValue && hasIndentedFollower(lines, i+1) {
			inBlockScalar = true
			out = append(out, line)
			continue
		}

		if newVal, ok := updates[key]; ok {
			ending := lineEnding(line)
			if newVal == "" && key == "epic" {
				continue
			}
			rewritten := rewriteScalarValue(text, key, newVal) + ending
			out = append(out, []byte(rewritten))
			continue
		}
		out = append(out, line)
	}

	eol := "\n"
	if len(lines) > 0 {
		eol = lineEnding(lines[0])
		if eol == "" {
			eol = "\n"
		}
	}
	for _, kv := range inserts {
		if kv.value == "" {
			continue
		}
		out = append(out, []byte(kv.key+": "+yamlScalar(kv.value)+eol))
	}

	var buf bytes.Buffer
	for _, l := range out {
		buf.Write(l)
	}
	return buf.Bytes(), nil
}

func splitLinesKeepEnding(b []byte) [][]byte {
	var out [][]byte
	for len(b) > 0 {
		i := bytes.IndexByte(b, '\n')
		if i < 0 {
			out = append(out, b)
			break
		}
		out = append(out, b[:i+1])
		b = b[i+1:]
	}
	return out
}

func stripLineEnding(line []byte) string {
	s := string(line)
	s = strings.TrimRight(s, "\n")
	s = strings.TrimRight(s, "\r")
	return s
}

func lineEnding(line []byte) string {
	s := string(line)
	if strings.HasSuffix(s, "\r\n") {
		return "\r\n"
	}
	if strings.HasSuffix(s, "\n") {
		return "\n"
	}
	return ""
}

// isTopLevelKeyLine returns true if line begins at column 0 with
// "<key>:" (followed by space, end of line, or a value).
func isTopLevelKeyLine(line string) bool {
	if line == "" || line[0] == ' ' || line[0] == '\t' || line[0] == '#' || line[0] == '-' {
		return false
	}
	colon := strings.IndexByte(line, ':')
	if colon <= 0 {
		return false
	}
	for i := 0; i < colon; i++ {
		c := line[i]
		if c == ' ' || c == '\t' {
			return false
		}
	}
	if colon+1 == len(line) {
		return true
	}
	c := line[colon+1]
	return c == ' ' || c == '\t' || c == '\r'
}

// parseTopLevelScalarLine extracts (key, value, hasValue) for a
// top-level "<key>: <value>" line. Returns isTopLevel=false for
// indented lines, comments, list items, or "<key>:" headers that open
// a block (no inline value).
func parseTopLevelScalarLine(line string) (key, value string, hasValue, isTopLevel bool) {
	if !isTopLevelKeyLine(line) {
		return "", "", false, false
	}
	colon := strings.IndexByte(line, ':')
	key = line[:colon]
	rest := line[colon+1:]
	rest = strings.TrimLeft(rest, " \t")
	if rest == "" {
		return key, "", false, true
	}
	return key, rest, true, true
}

// hasIndentedFollower reports whether the next non-blank line at or
// after start is indented (i.e. continues a block list, block
// mapping, or block scalar started by a "<key>:" header at the
// previous line). Returns false at EOF or if the next non-blank line
// is itself a top-level key — in that case the bare "<key>:" was an
// empty scalar, not a block header.
func hasIndentedFollower(lines [][]byte, start int) bool {
	for j := start; j < len(lines); j++ {
		text := stripLineEnding(lines[j])
		if text == "" {
			continue
		}
		return text[0] == ' ' || text[0] == '\t'
	}
	return false
}

// rewriteScalarValue replaces the value portion of a "<key>: <value>"
// line, preserving the leading "<key>: " prefix exactly. Trailing
// inline comments are dropped along with the old value because the
// YAML decoder doesn't surface them anyway, so we'd be guessing.
func rewriteScalarValue(line, key, newVal string) string {
	colon := strings.IndexByte(line, ':')
	prefix := line[:colon+1]
	rest := line[colon+1:]
	leadingWS := ""
	for len(rest) > 0 && (rest[0] == ' ' || rest[0] == '\t') {
		leadingWS += string(rest[0])
		rest = rest[1:]
	}
	if leadingWS == "" {
		leadingWS = " "
	}
	return prefix + leadingWS + yamlScalar(newVal)
}

// yamlScalar formats v as a YAML scalar safe for emission on a
// "<key>: <value>" line. Values that need quoting (containing :, #,
// leading/trailing whitespace, etc.) get single-quoted; everything
// else is bare. Mirrors yaml.v3's plain-scalar style.
func yamlScalar(v string) string {
	if v == "" {
		return "''"
	}
	if needsQuoting(v) {
		return "'" + strings.ReplaceAll(v, "'", "''") + "'"
	}
	return v
}

func needsQuoting(v string) bool {
	if v == "" {
		return true
	}
	if v[0] == ' ' || v[len(v)-1] == ' ' {
		return true
	}
	switch v[0] {
	case '!', '&', '*', '{', '[', '|', '>', '%', '@', '`', '"', '\'':
		return true
	}
	for i := 0; i < len(v); i++ {
		c := v[i]
		if c == '#' || c == ':' || c == '\n' || c == '\r' || c == '\t' {
			return true
		}
	}
	switch strings.ToLower(v) {
	case "true", "false", "yes", "no", "null", "~", "on", "off":
		return true
	}
	return false
}
