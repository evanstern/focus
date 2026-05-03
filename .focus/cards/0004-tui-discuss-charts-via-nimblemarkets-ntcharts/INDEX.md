---
schema_version: 2
id: 4
uuid: 019def84-255e-79f2-92bf-1d4e7c758c17
title: 'TUI: discuss charts via NimbleMarkets/ntcharts'
type: card
status: backlog
priority: p2
project: focus
created: 2026-05-03
tags: [tui, design]
---

## Summary

Decide whether `NimbleMarkets/ntcharts` (terminal charting library)
deserves a place in focus.

This is a **design-discussion card**, not ready-to-build. iris's
pre-read leans **pass for now**; the discussion is whether that's right.

## Recommendation (iris pre-read)

**Pass for now.** Revisit if focus's scope expands.

### What ntcharts is

A terminal charting library for Bubble Tea. v2.1.0 as of 2026-05-01.
13 chart types: Canvas, Bar, Line, Time Series, Streamline, Waveline,
Scatter, Heat Map, OHLC/Candle, Sparkline, Picture, Chart Picture,
Heat Picture. Render-only models (you embed in your own
`tea.Model`, call `Update`+`Draw`+`View` manually).

Production users: Weights & Biases (ML metrics TUI, ~1200-line
integration), Noumena Network's `nmon` (system monitoring). 708 stars,
actively maintained.

Module: `github.com/NimbleMarkets/ntcharts/v2`.

### Why this would be cool in focus

Plausibly useful chart shapes for a kanban tool:

| chart | maps to | value |
|---|---|---|
| Time Series | burndown — cards remaining over time | high if you actually look at it |
| Streamline | cards-per-day cumulative | some |
| Time Series | WIP-over-time (active count) | some |
| Bar | cycle time histogram (created → done) | nice trivia |
| Heat Map | status x date heatmap | awkward fit |

A `focus chart burndown` subcommand or a new "stats" pane in the TUI
could surface these.

### Why pass

1. **Persistence cost.** Every chart needs time-series data. Focus is
   stateless per-invocation; we don't write daily snapshots. Adopting
   ntcharts means adopting `.focus/snapshots.json` (or similar) +
   a snapshot-on-write hook in every mutating command. That's
   non-trivial state for a feature most solo users won't open twice.

2. **Layout pressure.** No natural place for charts in the current
   TUI. Status bar is one line; nav + preview already fill the
   body; help is an overlay. We'd need a third view mode and a way
   to switch into it. The build-cost-to-payoff ratio is bad for a
   tool that ships its hello-world experience as `focus new` →
   `focus board` → `focus done`.

3. **Signal-to-noise on a single-user board.** ntcharts shines on
   continuous dense data (system metrics, ML training curves).
   A kanban with maybe 5 transitions per week renders as a flat
   line on most chart types. The chart itself is the cool part,
   not the insight.

4. **Design ethos conflict.** focus's stated principles
   (`designs/focus-v2.md`): "opinions over machinery, no daemon,
   every invocation short-lived." Persistent time-series storage
   is exactly the machinery the design says no to. A chart needs
   N invocations of focus to mean anything; that breaks the
   "every invocation short-lived" loop.

### What would change my mind

- focus picks up multi-user / shared-board features (the design
  doc says no, but that could change). Trends across a team are
  legitimately useful.
- Someone wants a `focus stats` subcommand badly enough to write
  the snapshot mechanism themselves. At that point ntcharts is
  the right rendering library.

### Lighter alternative if charts ARE worth it

`asciigraph` (github.com/guptarohit/asciigraph). Zero deps, simple
line graphs, no Bubble Tea integration. Better fit for a one-shot
`focus burndown` CLI than ntcharts' interactive Bubble Tea models.
Trade-off: no interactivity, no mouse, no other chart types.

## Open questions for the discussion

1. Do you actually look at burndown charts? Honest answer changes
   the calculus.
2. If yes — is `focus burndown` the shape (one-shot CLI image), or
   is `focus tui --view stats` (full interactive pane) the shape?
3. Snapshot strategy if we go that way: snapshot on every
   transition (high write load) vs. once-a-day cron-style (drift)
   vs. derive from index.json's `updated` field if we add one.
   The design doc explicitly removed `updated` — undoing that has
   downstream implications.

## Out of scope (for the binary)

If we want any of this, it could equally well live as an
**external script** that reads `.focus/index.json` periodically and
plots — keeping focus the binary clean. The orchestrator (iris)
or a wrapper could own this without touching focus's code.

## Done when

- Decision made: pass / adopt / external-script-only.
- If adopt: build cards filed for snapshot mechanism + chart view.
- This card is killed.

## Evidence

- ntcharts repo: https://github.com/NimbleMarkets/ntcharts (v2.1.0)
- wandb integration: github.com/wandb/wandb/blob/main/core/internal/leet/epochlinechart.go
- asciigraph (lighter alt): https://github.com/guptarohit/asciigraph
- focus design doc § "Out of scope": no daemon, no time tracking,
  no notifications.

## Provenance

Filed by iris 2026-05-03 at Zach's request. Pre-read leans pass;
discussion happens in chat and updates this card as decisions land.
</content>
