package board

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/evanstern/focus/internal/board/card"
)

// snapshot returns sha256 hashes of every regular file under dir.
// Used to spot which paths a state-mutating command touches.
func snapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		h := sha256.Sum256(data)
		rel, _ := filepath.Rel(root, path)
		out[rel] = fmt.Sprintf("%x", h)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return out
}

// changedPaths returns the rel paths whose hash differs between
// before and after, sorted for deterministic output.
func changedPaths(before, after map[string]string) []string {
	var out []string
	for p, h := range after {
		if before[p] != h {
			out = append(out, p)
		}
	}
	for p := range before {
		if _, ok := after[p]; !ok {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}

// TestStateMutationsTouchOnlyTargetCard exercises every state-mutating
// command (kill, done, activate, park, revive) and asserts exactly
// two paths change: the targeted card's INDEX.md and .focus/index.json.
// The bug this guards against (#0008) showed killing #0004 silently
// flipping #0002's status because of stale in-memory state.
func TestStateMutationsTouchOnlyTargetCard(t *testing.T) {
	cases := []struct {
		name string
		prep func(b *Board, id int)
		mut  func(b *Board, id int) error
	}{
		{
			"kill from backlog",
			func(b *Board, id int) {},
			func(b *Board, id int) error { _, err := b.Kill(id, false); return err },
		},
		{
			"activate from backlog",
			func(b *Board, id int) {},
			func(b *Board, id int) error { _, err := b.Activate(id, false); return err },
		},
		{
			"park from active",
			func(b *Board, id int) { _, _ = b.Activate(id, false) },
			func(b *Board, id int) error { _, err := b.Park(id, false); return err },
		},
		{
			"done from active",
			func(b *Board, id int) { _, _ = b.Activate(id, false) },
			func(b *Board, id int) error { _, err := b.Done(id, false); return err },
		},
		{
			"revive from archived",
			func(b *Board, id int) { _, _ = b.Kill(id, false) },
			func(b *Board, id int) error { _, err := b.Revive(id, false); return err },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := setupBoard(t)
			var ids []int
			for i := 0; i < 5; i++ {
				c, _, _ := b.NewCard(fmt.Sprintf("card-%d", i), NewCardOpts{})
				ids = append(ids, c.ID)
			}
			target := ids[2]
			tc.prep(b, target)

			before := snapshot(t, b.Root)
			if err := tc.mut(b, target); err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			after := snapshot(t, b.Root)

			changed := changedPaths(before, after)
			dir, _ := b.FindCardDir(target)
			wantTarget := filepath.Join(".focus", "cards", dir, "INDEX.md")
			wantIndex := filepath.Join(".focus", "index.json")

			if len(changed) != 2 {
				t.Errorf("%s: expected exactly 2 paths to change, got %d:\n  %s",
					tc.name, len(changed), strings.Join(changed, "\n  "))
			}
			hasTarget := false
			hasIndex := false
			for _, p := range changed {
				switch p {
				case wantTarget:
					hasTarget = true
				case wantIndex:
					hasIndex = true
				default:
					t.Errorf("%s: unexpected path changed: %s", tc.name, p)
				}
			}
			if !hasTarget {
				t.Errorf("%s: target card %s not modified", tc.name, wantTarget)
			}
			if !hasIndex {
				t.Errorf("%s: %s not modified", tc.name, wantIndex)
			}
		})
	}
}

// TestStatusFlipIsOneLineDiff exercises the round-trip preservation
// contract from #0009: a status flip on a card with mixed-style
// frontmatter and a hand-wrapped body must produce a one-line diff.
func TestStatusFlipIsOneLineDiff(t *testing.T) {
	original := `---
schema_version: 2
id: 4
uuid: 019dfa36-aaaa-bbbb-cccc-ddddeeeeffff
title: 'Mixed style frontmatter'
type: card
status: backlog
priority: p2
project: focus
created: 2026-05-05
tags: [tui, design]
---
## Summary

Hand-wrapped paragraph one with several
short lines that focus must not reflow
when only the status changes.

Hand-wrapped paragraph two also has
multiple lines.
`

	c, err := card.Parse([]byte(original))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	c.Status = card.StatusArchived

	out, err := card.MarshalUpdate(c)
	if err != nil {
		t.Fatalf("MarshalUpdate: %v", err)
	}

	origLines := strings.Split(original, "\n")
	outLines := strings.Split(string(out), "\n")
	if len(origLines) != len(outLines) {
		t.Fatalf("line count drifted: orig=%d out=%d\nout:\n%s",
			len(origLines), len(outLines), out)
	}
	diffs := 0
	for i := range origLines {
		if origLines[i] != outLines[i] {
			diffs++
			t.Logf("line %d: %q → %q", i+1, origLines[i], outLines[i])
		}
	}
	if diffs != 1 {
		t.Errorf("expected exactly 1 changed line, got %d", diffs)
	}
}

// TestRoundTripNoEditByteIdentical exercises the focus-edit-no-edits
// contract: parsing then re-marshaling without any field changes
// must produce byte-identical output.
func TestRoundTripNoEditByteIdentical(t *testing.T) {
	original := `---
schema_version: 2
id: 7
uuid: 019dfa36-eeee-ffff-0000-111122223333
title: Normal card
type: card
status: backlog
priority: p1
project: focus
created: 2026-05-05
tags: [bug, cli]
contract:
  - First contract item
  - Second contract item
---
## Summary

Body line one.
Body line two unchanged by save.

Final paragraph.
`
	c, err := card.Parse([]byte(original))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	out, err := card.MarshalUpdate(c)
	if err != nil {
		t.Fatalf("MarshalUpdate: %v", err)
	}
	if string(out) != original {
		t.Errorf("non-mutating round trip changed bytes\nWANT:\n%s\nGOT:\n%s", original, out)
	}
}

// TestRoundTripPreservesBlockTags ensures tags written as a block
// list stay a block list across a status flip.
func TestRoundTripPreservesBlockTags(t *testing.T) {
	original := `---
schema_version: 2
id: 8
uuid: 019dfa36-bbbb-cccc-dddd-eeeeffff0000
title: Block-list tags
type: card
status: backlog
priority: p2
project: focus
created: 2026-05-05
tags:
  - tui
  - design
---
Body.
`
	c, err := card.Parse([]byte(original))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	c.Status = card.StatusArchived
	out, err := card.MarshalUpdate(c)
	if err != nil {
		t.Fatalf("MarshalUpdate: %v", err)
	}
	if !strings.Contains(string(out), "tags:\n  - tui\n  - design") {
		t.Errorf("block-list tags lost flow style:\n%s", out)
	}
	if strings.Contains(string(out), "tags: [") {
		t.Errorf("block-list tags collapsed to inline:\n%s", out)
	}
}

// TestSetBodyPreservesFrontmatter exercises the SetBody op (used by
// the MCP focus_edit_body tool): swapping the body must leave
// frontmatter byte-identical.
func TestSetBodyPreservesFrontmatter(t *testing.T) {
	b := setupBoard(t)
	c, _, _ := b.NewCard("test", NewCardOpts{})
	dir, _ := b.FindCardDir(c.ID)
	path := filepath.Join(b.CardsDir(), dir, "INDEX.md")

	beforeBytes, _ := os.ReadFile(path)
	beforeFM := frontmatterOf(t, beforeBytes)

	if err := b.SetBody(c.ID, "## New body\n\nWith content.\n"); err != nil {
		t.Fatalf("SetBody: %v", err)
	}

	afterBytes, _ := os.ReadFile(path)
	afterFM := frontmatterOf(t, afterBytes)

	if beforeFM != afterFM {
		t.Errorf("SetBody changed frontmatter\nBEFORE:\n%s\nAFTER:\n%s", beforeFM, afterFM)
	}
	if !strings.Contains(string(afterBytes), "## New body") {
		t.Errorf("SetBody did not write body:\n%s", afterBytes)
	}
}

// TestRoundTripPreservesCRLF guards against a bug where MarshalUpdate
// hardcoded "---\n" delimiters and produced mixed line endings on
// CRLF-saved cards (Parse explicitly tolerates CRLF input).
func TestRoundTripPreservesCRLF(t *testing.T) {
	original := "---\r\nschema_version: 2\r\nid: 11\r\nuuid: 019dfa36-1111-2222-3333-444455556666\r\ntitle: CRLF card\r\ntype: card\r\nstatus: backlog\r\npriority: p2\r\nproject: focus\r\ncreated: 2026-05-05\r\n---\r\nBody line one.\r\nBody line two.\r\n"

	c, err := card.Parse([]byte(original))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	c.Status = card.StatusArchived
	out, err := card.MarshalUpdate(c)
	if err != nil {
		t.Fatalf("MarshalUpdate: %v", err)
	}
	if strings.Contains(string(out), "\r\n") == false {
		t.Errorf("CRLF lost on save:\n%q", out)
	}
	if strings.Contains(string(out), "---\n") && !strings.Contains(string(out), "---\r\n") {
		t.Errorf("delimiter line ending downgraded to LF on a CRLF card:\n%q", out)
	}
}

// TestEmptyScalarValueRewritten guards against a bug where a top-level
// "<key>:" with no inline value got treated as a block header and the
// update was silently dropped. Real example: a card with `epic:`
// (literal empty scalar) being assigned an epic should rewrite the
// line to `epic: <id>`, not skip it.
func TestEmptyScalarValueRewritten(t *testing.T) {
	original := `---
schema_version: 2
id: 12
uuid: 019dfa36-2222-3333-4444-555566667777
title: Empty scalar
type: card
status: backlog
priority: p2
project: focus
created: 2026-05-05
owner:
---
Body.
`
	c, err := card.Parse([]byte(original))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	c.Owner = "ash"
	out, err := card.MarshalUpdate(c)
	if err != nil {
		t.Fatalf("MarshalUpdate: %v", err)
	}
	if !strings.Contains(string(out), "owner: ash") {
		t.Errorf("empty scalar not rewritten:\n%s", out)
	}
}

func frontmatterOf(t *testing.T, data []byte) string {
	t.Helper()
	parts := strings.SplitN(string(data), "\n---\n", 2)
	if len(parts) != 2 {
		t.Fatalf("no closing frontmatter delimiter in:\n%s", data)
	}
	return parts[0]
}
