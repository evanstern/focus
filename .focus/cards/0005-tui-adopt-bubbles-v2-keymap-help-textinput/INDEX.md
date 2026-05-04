---
schema_version: 2
id: 5
uuid: 019df0bf-0eac-7aa2-b741-2225d03070bd
title: 'TUI: adopt bubbles v2 — KeyMap + help + textinput'
type: card
status: archived
priority: p1
project: focus
created: 2026-05-04
contract:
  - All keybindings live in a single KeyMap struct of bubbles/key.Binding values
  - The router uses key.Matches against the KeyMap (no string-literal key matches in keys.go)
  - The help pane is rendered by bubbles/help from the KeyMap, not hand-written
  - 'The : and / mode buffers are bubbles/textinput.Model, not hand-rolled'
  - 'paste, ctrl-w (delete word), ctrl-a/e (line edges) work in : and / mode'
  - All bubbles + bubbletea + lipgloss imports use the charm.land v2 module paths
  - go.mod pinned to bubbletea v2.x, bubbles v2.x, lipgloss v2.x
  - Existing tests still pass; new tests cover key.Matches dispatch and help auto-generation
  - 'wiki/decisions/focus-tui-keybinds.md note: source of truth becomes the KeyMap struct, help_view.go is gone'
tags:
  - tui
  - refactor
---

## Summary

Adopt three components from `bubbles` in a single PR, and bump the entire TUI's bubbles/bubbletea/lipgloss imports to v2 in the same swing.

This is the build card for the discussion captured in #0003. Decisions below are locked; #0003 will be killed once this card moves.

## Locked decisions (from #0003 discussion)

| Decision | Choice |
|---|---|
| What to adopt now | `key.Binding`, `help`, `textinput` |
| Sequencing | Single PR for all three |
| v1 vs v2 module paths | Bump to v2 (`charm.land/bubbles/v2` etc.) in this same PR |
| What to skip | `list`, `table`, `paginator`, `spinner`, `progress`, `timer`, `stopwatch`, `filepicker`, `cursor` |
| What to defer | `textarea` (only useful if we add in-TUI editing; today `e` shells to $EDITOR, which is correct) |

## Concrete shape

### KeyMap struct

```go
// internal/tui/keys.go (or new internal/tui/keymap.go)

type KeyMap struct {
    // Pane focus
    FocusNext, FocusPrev key.Binding

    // Nav movement
    Up, Down, Top, Bottom    key.Binding
    JumpDown, JumpUp         key.Binding  // ctrl-d / ctrl-u

    // Preview scroll
    ScrollDown, ScrollUp                key.Binding
    ScrollTop, ScrollBottom             key.Binding
    ScrollHalfPgDown, ScrollHalfPgUp    key.Binding
    ScrollPgDown, ScrollPgUp            key.Binding  // ctrl-f / ctrl-b

    // Filter cycle
    FilterNext, FilterPrev   key.Binding

    // Layout cycle
    LayoutCycle              key.Binding

    // Actions
    Edit, Activate, Park, Done, Kill, Revive  key.Binding

    // Modes
    Search, Command, Help, Quit  key.Binding
}

func DefaultKeyMap() KeyMap {
    return KeyMap{
        FocusNext: key.NewBinding(
            key.WithKeys("tab"),
            key.WithHelp("tab", "focus next pane"),
        ),
        // ...
    }
}
```

The router becomes `key.Matches(msg, m.keys.Up)` instead of `switch msg.String() { case "k", "up": }`.

### help auto-generated

```go
// internal/tui/help_view.go gets deleted entirely.

// In the model:
m.help = help.New()

// In View when viewHelp is active:
m.help.View(m.keys)
```

`bubbles/help` reads the `KeyMap` and renders short/full views. The shipped help text becomes a function of the keymap — drift is structurally impossible.

### textinput in : and / mode

Replace `searchState{query string}` and `commandState{input string}` with `textinput.Model` instances. Wire KeyMsg into `Update()`, read `Value()` on enter.

The hand-rolled `isPrintable()` helper goes away.

## v1 → v2 import bumps

| from | to |
|---|---|
| `github.com/charmbracelet/bubbletea` | `charm.land/bubbletea/v2` |
| `github.com/charmbracelet/bubbles/viewport` | `charm.land/bubbles/v2/viewport` |
| `github.com/charmbracelet/lipgloss` | `charm.land/lipgloss/v2` |

Signature changes to expect (per the bubbles v2 upgrade guide):

- `tea.KeyMsg` → `tea.KeyPressMsg` in switch types
- Default keymap variables become functions (`DefaultKeyMap()`)
- Some component fields become getter/setter pairs (e.g. viewport)
- Functional options pattern for some constructors

If anything breaks unexpectedly, fix it in the same PR rather than splitting — the goal is one coherent migration.

## Out of scope

- `textarea` for in-TUI body editing — `e`/`enter`/`o` continue to shell out to `$EDITOR`. Filed as a separate consideration if we ever want it.
- Any other bubbles component.
- Refactoring keymap into a pluggable / config-driven shape. Static struct is fine for v0.x.

## Tests

- `key.Matches(KeyPressMsg{Type: KeyTab}, m.keys.FocusNext)` returns true; same for shift-tab + every other binding.
- A focused-pane test still passes (Tab cycles focus, j/k scrolls preview when preview is focused, etc.) — existing `internal/tui/focus_test.go` should run without changes.
- Help view renders without erroring; contains the FocusNext key and its description.
- `:new "title"` round-trips: type the chars, hit enter, verify the card title is `title`. (This is a real test of the textinput Update wire-up.)
- Paste a multi-byte string into / mode (simulate via `tea.KeyMsg` with multi-rune Runes); verify the textinput accepts it cleanly.

## Wiki update (post-merge)

Iris will update `wiki/decisions/focus-tui-keybinds.md` to point at the KeyMap struct as the source of truth and note that `help_view.go` is gone. The wiki page already has a "file wins on disagreement" disclaimer; that disclaimer now points at the keymap struct.

## Provenance

Filed by iris 2026-05-04 after the #0003 discussion landed in chat. #0003 will be killed once this card is in flight.

Don't start work on this card until card #0002 (CLI shell completions) merges — that PR is in-flight at `feature/focus-cli-completions` and we want a single rebase target on main.
</content>
