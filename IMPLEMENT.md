# Focus TUI Implementation — Card #19

## Status

Phase 1 (MVP) and Phase 2 (Polish) are **complete and in-tree**. `bin/focus` has the full `_focus_cmd_tui()` function wired into the dispatcher. All 36 tests pass (31 original + 5 TUI-specific).

## What's Done

- `_focus_cmd_tui()` — full interactive TUI with alternate screen buffer, raw input, cleanup traps
- **Tab bar** — 5 status tabs (active/backlog/done/parked/killed) with color + highlight
- **Card list** — scrollable, sorted by priority, shows id/priority/title/project, `>` cursor
- **Detail pane** — bottom split showing title, project, priority, status, updated, body preview
- **Navigation** — `j`/`k`/arrows for cursor, `h`/`l`/arrows/Tab for tabs
- **State transitions** — `a` activate, `p` park, `d` done, `K` kill (all call existing focus functions)
- **Edit** — `e` drops to `$EDITOR`, restores TUI on return
- **Search** — `/` enters search mode, filters cards by title/project, Esc clears
- **Resize** — SIGWINCH trap triggers full redraw
- **Terminal restore** — trap on EXIT/INT/TERM restores stty, rmcup, cursor visibility
- Dispatcher entry: `tui)  _focus_cmd_tui ;;`
- **Terminal size guard** — 80x24 minimum check on entry
- **Help/README/completions** — `tui` command documented everywhere
- **Empty tab message** — "(no cards)" centered in list area
- **Scroll indicator** — `[1-5 of 12]` right-aligned on tab bar
- **Message timeout** — status messages clear after 3 keypress cycles
- **`n` keybinding** — inline new card creation with title/project prompts
- **NO_COLOR support** — reverse-video skipped, `>` text cursor always visible
- **TUI tests** — 5 non-interactive tests (terminal check, command recognition, help/completions)

## What's Left

### Must Do — Complete

1. ~~**Update help text** — Add `tui` to `_focus_cmd_help()` and the commands table~~
2. ~~**Update README.md** — Add `tui` to the commands table~~
3. ~~**Update completions** — Add `tui` to both bash and zsh completion command lists~~
4. ~~**Write tests** — Non-interactive tests for the TUI~~
5. ~~**Minimum terminal size guard** — Check `tput lines`/`tput cols` >= 80x24 on entry, show error if too small~~

### Should Do (Polish) — Complete

6. ~~**Empty tab message** — Show "(no cards)" centered when a tab has zero cards~~
7. ~~**Scroll indicator** — Show `[1-5 of 12]` right-aligned on tab bar when list is scrolled~~
8. ~~**Message timeout** — Clear `_tui_message` after 3 keypress cycles~~
9. ~~**`n` keybinding** — Inline new card creation (prompt for title/project in status bar)~~
10. ~~**Color in NO_COLOR mode** — Reverse-video skipped in NO_COLOR; `>` text cursor marker always shown~~

### Nice to Have

11. **Enter to expand** — Full-screen card detail view (the spec mentions this)
12. **Contract status** — Show contract progress (2/5) in detail pane
13. **WIP limit warning** — Flash status bar when activate would exceed WIP

## Architecture Notes

- All TUI code lives in `bin/focus` between the `# -- TUI --` and `# -- Main Dispatcher --` comment blocks (~550 lines)
- TUI state is local variables inside `_focus_cmd_tui()`: `_tui_tab`, `_tui_cursor`, `_tui_scroll`, `_tui_search`, `_tui_search_mode`, `_tui_search_buf`, `_tui_message`
- Card data is loaded into parallel arrays (`_tui_card_ids`, `_tui_card_titles`, etc.) by `_focus_tui_load_cards()`
- Drawing is split: `_focus_tui_full_draw()` redraws everything; `_focus_tui_partial_draw()` skips header+tabs (used for cursor movement)
- Layout proportions: card list gets ~65% of available rows, detail pane ~35%
- State transitions (a/p/d/K) call the existing `_focus_cmd_*` functions, then reload cards and redraw
- The `d` (done) action bypasses contract prompts by directly setting fields (can't do interactive prompts inside raw mode)

## Testing the TUI Manually

```sh
# From this worktree:
./bin/focus tui

# Or install and run:
make install PREFIX=~/.local
focus tui
```

## Contract (from card #19)

- [x] focus tui launches and renders without errors
- [x] Tab bar navigates between all 5 statuses
- [x] Card list shows correct cards per status, sorted by priority
- [x] Detail pane displays selected card metadata + body
- [x] Keyboard shortcuts trigger state transitions (activate/park/done/kill)
- [x] Edit shortcut opens $EDITOR and returns to TUI
- [x] Search filters cards by title/project
- [x] Works on 80x24 minimum terminal size (guard added)
- [x] No external dependencies (pure bash + ANSI/tput)
- [x] Tests pass and existing focus tests unaffected (36/36 pass)
