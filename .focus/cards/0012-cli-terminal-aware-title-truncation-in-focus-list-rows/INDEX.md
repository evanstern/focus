---
schema_version: 2
id: 12
uuid: 019e0995-a56b-73b1-802f-e135c1e2a3dd
title: 'CLI: terminal-aware title truncation in focus list rows'
type: card
status: archived
priority: p2
project: focus
created: 2026-05-08
---

## Why

`focus list` and `focus board` use `%-40s` on the title field (see
`internal/cli/format.go:57`), which left-pads short titles to 40
chars but never truncates long ones. Result: a card with a 70-char
title shoves all subsequent columns rightward, breaking column
alignment for the whole list.

Reproduction:

```
$ focus list
archived   #0009  Bug: focus card save reformats body and tags (lossy markdown round-trip)  focus       p1    -
archived   #0010  TUI refresh on external changes (auto-watch, manual reload, or both)  focus       p2    -
backlog    #0011  Hardening for byte-preserving card save (review followup from PR #14)  focus       p2    -
```

The project / priority / owner columns drift right based on title
length. Every other unix tool (`git log --oneline`, `ls -l`, `ps`)
truncates the variable-width column to fit the terminal; we don't.

## Locked decisions

1. **Approach: terminal-aware truncation.** Detect terminal width
   at run time. Reserve fixed widths for the non-title columns
   (status, id, project, priority, owner). Give the title whatever
   is left. Truncate longer titles with a single-rune ellipsis
   `…` so the visible width is exact.

2. **Width detection.** Use `golang.org/x/term` (or
   `term.GetSize` from the stdlib equivalent already in tree —
   check before adding deps). On error / non-tty / pipe, fall
   back to a fixed 80-column assumption.

3. **Ellipsis character is `…` (U+2026), not `...`.** A single
   rune means visible width matches the truncation budget
   exactly. UTF-8 terminals are universal; this is not 1995.

4. **Minimum title budget.** If the terminal is narrower than
   the fixed columns + ~20 chars of title room, do not crash and
   do not produce negative-width truncation. Floor the title
   width at 20 and let the row wrap naturally on absurdly narrow
   terminals — that's the user's choice at that point.

5. **Escape hatch for piping: `--no-truncate` flag.** When
   piping or scripting, full titles are useful. The flag opts
   back into the current behavior. Default is truncated.

6. **Honor the existing principle, don't reverse it.** The
   current code's comment says "truncating titles silently is
   hostile." Truncation **with a clear `…` marker** isn't
   silent — `focus show <id>` is right there for the full
   title. Keep that comment, update it to explain the new
   policy.

7. **Scope: `focus list`, `focus board`, `focus epic list`.**
   All three use the same `formatRow` / `printEpicList` helpers.
   Apply the policy uniformly. `focus show` is unchanged
   (it already prints the full title).

## Done when

- [ ] `formatRow` (and `printEpicList`'s inline format) compute
      a title budget from terminal width minus fixed columns.
- [ ] Titles longer than the budget are truncated to
      `budget-1` runes plus `…`. Rune-count, not byte-count
      (utf8.RuneCountInString or equivalent).
- [ ] Width detection falls back to 80 on non-tty / pipe / error.
- [ ] Title budget is floored at 20 runes for absurdly narrow
      terminals.
- [ ] `--no-truncate` flag on `focus list` and `focus board`
      restores full-width titles. (Pipe a tabular column-aligned
      output is then the caller's problem.)
- [ ] Test asserts: 80-col terminal → predictable column
      positions; 40-col terminal → minimum budget honored;
      `--no-truncate` → full title preserved; non-ASCII title
      truncates by rune count, not bytes.
- [ ] Comment in `format.go` updated: keep the principle
      ("silent truncation is hostile"), explain why the new
      policy isn't silent (`…` is the marker).
- [ ] Pre-flight muscle: tree parity on `.focus/cards/`,
      `IMPLEMENT.md` deleted before PR open, `go install
      ./cmd/focus` step zero on local verify.

## Out of scope

- Terminal-width detection for the TUI (the TUI already handles
  this via Bubble Tea).
- Changing the column order or adding/removing columns.
- Color in CLI output (still v0.1.0 stance: no color, styling
  lives in the TUI).
- A second pass for `focus show` (it's already fine).

## Notes

Discussion that led here is in iris's 2026-05-08 memory. The
short version: original suggestion was a fixed 100-char minimum
row. Pushed back because 100 chars is wider than typical
terminals, especially split panes; convention is fit-the-terminal
truncate. Zach picked the terminal-aware approach.
