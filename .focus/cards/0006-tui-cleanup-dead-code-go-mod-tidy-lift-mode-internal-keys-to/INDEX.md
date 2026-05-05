---
schema_version: 2
id: 6
uuid: 019df141-1cdc-7360-bec3-2dbb47fd0412
title: 'TUI cleanup + fix card-save scope and round-trip bugs'
type: card
status: backlog
priority: p1
project: focus
created: 2026-05-04
tags: [tui, cleanup, bug, data-integrity]
contract:
  - "Bug fix (was #0008): state-mutating commands (kill/done/activate/park/revive) must only modify the targeted card's INDEX.md and .focus/index.json"
  - "Bug fix (was #0009): frontmatter mutations preserve the body byte-for-byte; tag-list flow style preserved (inline stays inline, block stays block); only the changed frontmatter key is rewritten"
  - "Test: with N cards, run a kill/done/activate/park/revive and assert exactly two paths appear in git status --porcelain (target INDEX.md + index.json)"
  - "Test: round-trip a card with mixed-style frontmatter and a hand-wrapped body through a status flip; assert the diff is exactly one line"
  - "Test: focus edit (no edits, save) produces a byte-identical file"
  - go mod tidy run; v1 bubbletea/lipgloss demoted from direct require
  - "internal/tui/keys.go: idForRow function removed (no callers)"
  - "internal/tui/keys.go: blank-identifier suppression `var _ = card.PaddedID` removed"
  - "tui.go:242 hand-rolled ctrl+c match replaced with key.Matches against KeyMap"
  - "esc/enter mode-internal escapes lifted to KeyMap entries (Cancel + Confirm bindings)"
  - viewHelp dismiss uses KeyMap entry (Dismiss or reuse Cancel) rather than literal string match
  - All existing tests still pass; no behavior change in TUI keybinds
---

## Summary

Bundles three workstreams into one PR:

1. **Card-save scope bug** (was #0008): state-mutating commands
   touch unrelated cards' frontmatter. P1 data-integrity bug.
2. **Card-save round-trip bug** (was #0009): mutations reformat
   the body and tag-list flow style, producing huge noisy diffs
   for one-line state changes. P1 author-intent bug.
3. **TUI cleanup follow-ups** from the PR #13 review (six P2
   items, was the original p3 chore on this card).

Single PR. All three workstreams. Bugfixes drive the priority
bump from p3 → p1.

## Recommended ordering

The bug fixes touch the card-save code path; the TUI cleanup
touches `internal/tui/`. They're independent in the diff. **Do
the bug fixes first** so the TUI cleanup PR's *own* INDEX.md
mutations (e.g. `focus done 6` after merge) don't produce noisy
diffs themselves.

Suggested commit order in the branch:
1. Fix card-save scope (#0008): state mutations only touch the
   target card.
2. Fix card-save round-trip (#0009): preserve body bytes and
   frontmatter flow style.
3. TUI cleanup items 1-6 below.

This is a recommendation, not a contract. One squashed commit
on merge either way.

## Part 1: Card-save bug fixes

### Bug 1 — kill rewrites unrelated cards (was #0008)

`focus kill 4` modified the frontmatter of an **unrelated** card
during a kill operation. Specifically, killing card #0004 flipped
card #0002's `status` from `archived` back to `backlog`.

Repro on commit `888eb6f` (v0.1.3), with #0002 archived and
#0004 in backlog:

```
$ git status   # clean
$ focus kill 4 --force
#0004 → archived
$ git status
modified:   .focus/cards/0002-cli-shell-completions-bash-zsh-fish/INDEX.md
modified:   .focus/cards/0004-tui-discuss-charts-via-nimblemarkets-ntcharts/INDEX.md
```

The #0002 diff was just `-status: archived` / `+status: backlog`.
Nothing else on #0002 changed.

**Why this matters.** A kill should be a one-card operation.
Touching any other card during kill is a write-amplification
bug. Already shipped this once — commit `7057ea5` on main
carried the damage; fixed in `aea3f68` immediately after.

**Likely root cause (hypothesis).** `focus kill` probably
rewrites every card's `INDEX.md` from the in-memory index, and
the in-memory state for #0002 was stale. Either (a) save scope
is too wide — kill is rewriting all cards, not just the target,
or (b) the index is hydrated from somewhere lossy that doesn't
read the on-disk frontmatter as source of truth. Probably the
same code path affects `done`, `park`, `activate`, `revive`.

**Done when:**
- `focus kill <id>` only modifies `.focus/cards/<dir>/INDEX.md`
  for the targeted card and `.focus/index.json`.
- Same guarantee for `activate`, `park`, `done`, `revive`.
- Test: with N cards, run a kill and assert exactly two paths
  appear in `git status --porcelain` (target's INDEX.md and
  the index.json).
- Test: any state-mutating command on card X must not change
  the on-disk content of any other card's INDEX.md.

### Bug 2 — body and tags reformatted on save (was #0009)

When focus mutates a card's frontmatter (e.g. via `kill`,
`activate`, `done`), the resulting INDEX.md is re-emitted with
different formatting than the input. Two observed effects on
card #0004 during the same `focus kill 4`:

1. Tags exploded from inline to block list:

   ```
   -tags: [tui, design]
   +tags:
   +  - tui
   +  - design
   ```

2. Body paragraphs unwrapped — lines that were hard-wrapped at
   ~70 chars in the source got concatenated into single long
   lines.

**Why this matters.** A status flip should be a one-line diff.
Reformatting turns it into a 50-line diff that hides the real
change. Round-trip lossiness blocks editor workflows: `focus
edit` → `:wq` → `focus done` will mangle the body even if you
didn't touch it.

**Likely root cause (hypothesis).** The save path probably
parses markdown into an AST or struct, then re-serializes from
that representation. Round-tripping markdown through any AST is
famously lossy. The fix is to **never re-serialize the body**.
Treat the body as opaque bytes; only rewrite the frontmatter
block.

**Implementation note.** Treat INDEX.md as a two-region file:

- **Frontmatter region:** parse + edit + re-serialize, but only
  the changed key is rewritten; all other keys preserved as
  bytes.
- **Body region:** pass through verbatim.

A YAML library that exposes the original token stream (rather
than parse-to-map-to-emit) is the right primitive for the
frontmatter region. If the current YAML library doesn't support
this, the cheapest fix is regex-based — find the `status:` line,
swap the value, leave everything else alone.

**Done when:**
- Frontmatter mutations preserve the body byte-for-byte.
- Frontmatter mutations preserve the *unmodified* frontmatter
  fields byte-for-byte (only the changed key is rewritten).
- Tags written as `[a, b]` stay `[a, b]`; tags written as a
  block list stay a block list.
- Test: write a card with mixed-style frontmatter and a
  hand-wrapped body; flip its status; assert the diff is
  exactly one line.
- Test: round-trip `focus edit` (no edits, save) produces a
  byte-identical file.

## Part 2: TUI cleanup (original #0006 scope)

Six items from the PR #13 review. Each shaves a small amount of
cruft or contract drift; none change behavior.

### 1. `go mod tidy`

After the v2 module bump, the direct-require block in `go.mod`
still has `bubbletea v1.3.10` and `lipgloss v1.1.1-0...` that
aren't directly imported (`go mod why` confirms neither is
needed by the main module). Running `go mod tidy` demotes them
to indirect and prunes a handful of unneeded indirect entries.

### 2. Remove `idForRow`

`internal/tui/keys.go:495` — function with no callers anywhere
in the codebase. Test helpers don't use it either. Delete.

### 3. Remove `var _ = card.PaddedID`

`internal/tui/keys.go:508` — blank-identifier import suppression
that's no longer needed. `card.PaddedID` is now legitimately
called in `board_view.go:349` and `preview.go:116`. Delete the
line.

### 4. Lift hand-rolled ctrl+c match into KeyMap

`internal/tui/tui.go:242`:

```go
case msg.Code == 'c' && msg.Mod == tea.ModCtrl:
    return m, tea.Quit
```

This duplicates the `Quit` binding's `"ctrl+c"` key. The intent
— make ctrl+c always quit, even from search/command/help modes
— is correct, but the implementation reaches around the KeyMap.
Two reasonable shapes:

**Option A:** Add a separate `ForceQuit` binding bound to just
`ctrl+c`, use `key.Matches(msg, m.keys.ForceQuit)`.

**Option B:** Check `key.Matches(msg, m.keys.Quit) && msg.Mod == tea.ModCtrl`.

Either keeps the literal out. Option A is clearer.

### 5. Lift `esc` and `enter` from `handleSearchKey` / `handleCommandKey`

`keys.go:323` and `keys.go:379` both have:

```go
switch msg.String() {
case "esc": ...
case "enter": ...
}
```

These are mode-internal escapes — esc cancels, enter confirms.
Defensible because textinput would otherwise consume them, but
the contract said "no string-literal key matches in keys.go"
and these are in keys.go.

Add to the KeyMap:

```go
Cancel  key.Binding  // esc
Confirm key.Binding  // enter
```

Replace the string-literal switches with `key.Matches`.

### 6. Lift `viewHelp` dismiss

`tui.go:263`:

```go
if s == "esc" || s == "q" {
    m.view = viewBoard
}
```

Reuse the `Cancel` binding from item 5 for esc, add a `Dismiss`
binding bound to just `q`.

## Tests (TUI portion)

Existing tests should pass unchanged. Add:

- `key.Matches` test for `Cancel` (esc), `Confirm` (enter),
  `Dismiss` (q in help) — same shape as the existing 28-case
  table in `keys_test.go`.
- Round-trip: enter search, type, hit `enter`, verify mode is
  normal and filter is preserved (already exists; should still
  pass).
- Round-trip: enter search, type, hit `esc`, verify mode is
  normal and filter is cleared (already exists).

## Out of scope

- Any behavior changes in the TUI. Structural only.
- Renaming existing bindings.
- Adding new commands (`:foo` etc.).
- A pretty-printer / canonicalizer subcommand. If users want
  reformat, that's an explicit `focus fmt` call, not a side
  effect of state mutations.
- The `--focus-dir` flag (#0007) — separate card.
- TUI refresh on external changes (#0010) — separate card.

## Linked / superseded

- #0008 (kill rewrites unrelated cards) — absorbed into Part 1
  above; archive on merge.
- #0009 (card save reformats body + tags) — absorbed into Part
  1 above; archive on merge.

## Provenance

Originally filed by iris 2026-05-04 (TUI cleanup chore) from PR
#13 review. Expanded 2026-05-05 to absorb #0008 and #0009 after
both bugs surfaced during the routine archival of #0004. Single
feature session, one PR, three contracts.
