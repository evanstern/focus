---
schema_version: 2
id: 3
uuid: 019def84-2550-7bd6-934d-a2b2f70fc4a6
title: 'TUI: discuss components from charmbracelet/bubbles'
type: card
status: backlog
priority: p1
project: focus
created: 2026-05-03
tags: [tui, design]
---

## Summary

Decide which `charmbracelet/bubbles` components are worth adopting in the
focus TUI. We already use `viewport` (since PR #11 — preview pane scroll).
This card scopes a discussion of the rest.

This is a **design-discussion card**, not ready-to-build. Convert to one or
more concrete build cards once the discussion lands.

## Recommendation (iris pre-read)

Adopt **2 components now**, **1 later**, **skip the rest**.

### Tier 1 — adopt soon

**1. `bubbles/key.Binding` — foundational.**
- Centralizes the keybind definitions. Currently we have string-literal key
  matches scattered across `internal/tui/keys.go`. A `KeyMap` struct of
  `key.Binding` values lets each binding carry its keys + help text in one
  place.
- Concrete shape:
  ```go
  type KeyMap struct {
      Up        key.Binding
      Down      key.Binding
      Activate  key.Binding
      Done      key.Binding
      // ...
  }
  var DefaultKeyMap = KeyMap{
      Up: key.NewBinding(
          key.WithKeys("k", "up"),
          key.WithHelp("k/↑", "move up"),
      ),
      // ...
  }
  ```
- Effort: medium. The router becomes `key.Matches(msg, DefaultKeyMap.Up)`.
- ROI: single source of truth; help-auto-generation becomes free.

**2. `bubbles/textinput` — replaces our hand-rolled `:`/`/` buffer.**
- Today `searchState` and `commandState` are bespoke string buffers with
  hand-handled backspace/printable-detection (`isPrintable`, `keys.go:191`).
  Works, but doesn't handle paste, wide-char input, ctrl-w word delete,
  ctrl-a/e line edges, etc. `textinput` does.
- Concrete shape: replace `m.search.query` with a `textinput.Model`. Forward
  KeyMsg into `Update()`. Read `Value()` on enter.
- Effort: low. Drop-in for the search + command-mode buffers.
- ROI: free editing affordances + fewer bugs.

### Tier 2 — adopt after `key.Binding` lands

**3. `bubbles/help` — auto-generates the help pane from the same key map.**
- Today `internal/tui/help_view.go` is hand-written and **already drifted
  from reality** during the v2 build (we caught and fixed in round 7).
  `bubbles/help` reads a `KeyMap` and renders short/full help, light/dark
  themed.
- Effort: medium. Requires the `key.Binding` refactor first.
- ROI: deletes ~80 lines of hand-rolled help text + drift becomes
  impossible.

### Tier "later if ever"

- **`textarea`** — only if we ever add an in-TUI `:edit` mode. Today `e`
  shells out to `$EDITOR`, which is the right call. Skip unless that
  changes.
- **`table`** — only if a future view shows tabular metadata. Not now.

### Skip entirely

- `list` — we have a hand-rolled nav with vim keys; `list`'s opinions
  (fuzzy filter, pagination UI) don't fit.
- `paginator` — viewport scrolling already covers this.
- `spinner`, `progress` — no async ops.
- `timer`, `stopwatch` — focus is not a time tracker.
- `filepicker` — no file-picker use case.
- `cursor` — used internally by textinput/textarea; no direct adoption.

## Open questions for the discussion

1. **Module path: `github.com/charmbracelet/bubbles` vs `charm.land/bubbles/v2`.**
   bubbles v2 lives at `charm.land/bubbles/v2`; v1 at the github path.
   We currently import `github.com/charmbracelet/bubbles/viewport`.
   What major are we on, and do we migrate everything to v2 in one
   sweep when we adopt new components?

2. **One PR per component, or one big "adopt key + textinput + help" PR?**
   - One big PR has the advantage that `help` gets adopted immediately
     after `key.Binding` lands, so we don't have a window where the help
     text is still hand-written against the new keymap.
   - One PR per component is more reviewable.
   - Recommend: bundle key + help in one PR (they're coupled), put
     textinput in its own.

3. **What's the upgrade path if v2 import paths bite us?**
   bubbles v2 bumped some signatures (e.g. `tea.KeyMsg` →
   `tea.KeyPressMsg` per the upgrade guide). Need to confirm bubbletea
   is also on a compatible major.

## Versions to pin

- bubbles v2.1.0 (March 2026) — latest stable.
- bubbletea v2 — must be on the matching major. Verify our `go.mod`.
- lipgloss v2 — same.

## Out of scope

- Forking bubbles components or wrapping them in our own abstractions.
  The whole point is using charm's discipline directly.
- Custom item delegates for `list` (we're skipping `list`).

## Done when

- We've decided which components to adopt and in what order.
- Build cards are filed for each adoption.
- This card is killed (its job is to gate the discussion, not to
  track the work).

## Provenance

Filed by iris 2026-05-03 at Zach's request. Pre-read above is iris's
recommendation; the discussion happens in chat and updates this card
as decisions land.
</content>
