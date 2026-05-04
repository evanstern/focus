---
schema_version: 2
id: 6
uuid: 019df141-1cdc-7360-bec3-2dbb47fd0412
title: 'TUI cleanup: dead code + go mod tidy + lift mode-internal keys to KeyMap'
type: card
status: backlog
priority: p3
project: focus
created: 2026-05-04
tags: [tui, cleanup]
contract:
  - go mod tidy run; v1 bubbletea/lipgloss demoted from direct require
  - "internal/tui/keys.go: idForRow function removed (no callers)"
  - "internal/tui/keys.go: blank-identifier suppression `var _ = card.PaddedID` removed"
  - "tui.go:242 hand-rolled ctrl+c match replaced with key.Matches against KeyMap"
  - "esc/enter mode-internal escapes lifted to KeyMap entries (Cancel + Confirm bindings)"
  - viewHelp dismiss uses KeyMap entry (Dismiss or reuse Cancel) rather than literal string match
  - All existing tests still pass; no behavior change
---

## Summary

Six P2 cleanup items surfaced during the PR #13 review. None
were blockers (the PR shipped clean), but each shaves a small
amount of cruft or contract drift.

This is a chore card, not a feature. Single PR, all six items.

## The list

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

This duplicates the `Quit` binding's `"ctrl+c"` key. The
intent — make ctrl+c always quit, even from search/command/help
modes — is correct, but the implementation reaches around the
KeyMap. Two reasonable shapes:

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

Either reuse the `Cancel` binding from item 5 (esc) plus a
`Dismiss` binding for q (or reuse Quit's q match — but that
also has ctrl+c, which already handles dismiss via the global
quit path). Simplest: reuse `Cancel` for esc, add a `Dismiss`
binding bound to just `q`.

## Tests

Existing tests should pass unchanged after this work. Add:

- `key.Matches` test for `Cancel` (esc), `Confirm` (enter),
  `Dismiss` (q in help) — same shape as the existing 28-case
  table in `keys_test.go`
- Round-trip test: enter search, type, hit `enter`,
  verify mode is normal and filter is preserved (already
  exists; should still pass)
- Round-trip test: enter search, type, hit `esc`, verify mode
  is normal and filter is cleared (already exists)

## Out of scope

- Any behavior changes. This is purely structural.
- Renaming existing bindings.
- Adding new commands (`:foo` etc.).

## Provenance

Filed by iris 2026-05-04 from the PR #13 review. Not blocking
any current work; pick up when convenient.
</content>
