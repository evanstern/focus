---
schema_version: 2
id: 13
uuid: 019e09ed-2a27-73e1-a692-e8e03eadfea1
title: 'CLI: use display-column width instead of rune count for CJK / wide-emoji alignment'
type: card
status: backlog
priority: p2
project: focus
created: 2026-05-08
---

## Why

Follow-up from PR #17 / card #0012. PR #17 shipped terminal-aware
truncation with rune-count math throughout. That's strictly better
than the previous byte-count behavior, but it's still wrong for
CJK and wide-emoji content: `世` is 1 rune but 2 terminal columns
in every common monospace font. So a row with a CJK title or
project name renders ~2x wider than an ASCII row with the same
rune count, and columns visibly drift across rows.

Reproduction (against current main after PR #17 ships):

```
$ stty cols 100
$ # board with mixed ASCII / CJK / emoji titles in /tmp/x
$ focus list
backlog    #0001  ASCII title goes here for comparison      focus       p2    -
backlog    #0002  world peace 世界平和 in Japanese            focus       p2    -
backlog    #0003  wide emoji title 🚀🚀🚀🚀🚀 here              focus       p2    -
                                                            ^ project col drifts
```

The `focus` column lands at three different on-screen positions
because rune count != display column count for the title segment.

## Locked decisions

1. **Switch all layout math from `utf8.RuneCountInString` to
   `runewidth.StringWidth`** (`github.com/mattn/go-runewidth`).
   Already an indirect dep via lipgloss; promote to direct in
   `go.mod`. No new transitive cost.

2. **Sites to convert in `internal/cli/format.go`:**
   - `padRunes` — measure with `runewidth.StringWidth`, not
     `utf8.RuneCountInString`. Rename to `padDisplayWidth`.
   - `truncateRunes` — accumulate display width, not rune
     count, when walking the string. Rename to
     `truncateDisplayWidth`. The `…` itself is 1 display
     column, so the math stays clean.
   - `formatRowWidth` — budget computation should subtract
     `runewidth.StringWidth(progress)` etc. instead of rune
     count. Project truncation uses display-width.
   - `printEpicList` — same treatment for its inline budget
     math (or reuse the shared helper).

3. **Test the actual claim.** Tests must assert
   **display-column equality**, not rune-count equality, when
   verifying alignment. Add a `displayWidth` test helper.

4. **Test matrix:**
   - CJK title (e.g. `世界平和`) at 80-col → assert display-
     column position of project column equals an ASCII row's
     project column position.
   - Wide-emoji title (e.g. `🚀🚀🚀`) at 80-col → same.
   - Combining-mark title (e.g. `é` as `e` + U+0301) → at
     least don't crash; visual alignment is best-effort.
   - ZWJ emoji cluster → same: don't crash, best-effort.

5. **Update the comment in `format.go`** that currently claims
   "non-ASCII titles (emoji, accents) cut at character
   boundaries" — qualify it. Combining marks split mid-grapheme
   today; that's acceptable but should be documented.

6. **Standing pre-flight:** tree parity on `.focus/cards/`,
   IMPLEMENT.md deleted before PR open, `go install
   ./cmd/focus` step zero on local verify.

## Done when

- [ ] `runewidth` promoted to direct dep in `go.mod`.
- [ ] All layout math in `internal/cli/format.go` uses
      `runewidth.StringWidth` for measurement and a display-
      width-aware truncate loop.
- [ ] Tests assert display-column-equality (not rune-count-
      equality) on CJK and wide-emoji titles.
- [ ] Mixed ASCII / CJK / emoji `focus list` output renders
      with vertically-aligned columns at any sane terminal
      width.
- [ ] `printEpicList` uses the shared display-width helpers.
- [ ] Comment updated to be honest about combining-mark /
      ZWJ-cluster best-effort behavior.
- [ ] Pre-flight muscle: tree parity, IMPLEMENT.md deleted,
      `go install` step zero.

## Out of scope

- Combining-mark / ZWJ-cluster perfection. Best-effort: don't
  crash, don't tear UTF-8, accept that some clusters split mid-
  glyph. A real fix needs `uniseg` and grapheme-cluster
  iteration; that's a bigger card if anyone files it.
- Reordering or adding/removing CLI columns.
- Color in CLI output (still no).

## Notes

This is a defect-of-degree from PR #17, not a regression. PR #17
shipped strictly-better-than-main alignment behavior; this card
finishes the job for non-ASCII content. Filed because it's the
honest follow-up and `runewidth` is one import away in the dep
tree already.
