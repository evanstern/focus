---
schema_version: 2
id: 10
uuid: 019dfa36-8e4a-7b31-aa78-bb6fad62da39
title: TUI refresh on external changes (auto-watch, manual reload, or both)
type: card
status: backlog
priority: p2
project: focus
created: 2026-05-05
tags: [tui, design]
---

## Summary

When the TUI is running and an external process changes the board
(another `focus` invocation, an editor saving an INDEX.md, an
agent calling the MCP server), the TUI doesn't notice. The user
sees stale state until they re-launch.

This is a **design-discussion card.** Three plausible shapes
with different cost/feel trade-offs.

## Three options

### A. Manual reload only

A `KeyMap.Reload` binding (default `r` or `F5`) re-reads
`.focus/` from disk and rebuilds the model.

- **Pro:** Trivial to implement. Predictable. No background
  work, no surprising mid-action redraws.
- **Pro:** Fits the "every invocation short-lived, no daemon"
  ethos of the design doc — the TUI is just a longer
  invocation, reload is explicit.
- **Con:** User has to know to press it. Stale state is
  invisible until reload.

### B. Auto-watch via fsnotify

Watch `.focus/cards/**/INDEX.md` and `.focus/index.json`. On
change events, debounce ~200ms then reload.

- **Pro:** Just works. Edit a card in another terminal, see
  it update. Agents using the MCP show up live.
- **Con:** fsnotify dep + per-platform quirks (macOS coalesces
  events, Linux inotify watch limits, Windows can be flaky).
- **Con:** Cursor/scroll/filter state must be preserved across
  reload — non-trivial UX work.
- **Con:** Mid-action redraws (typing in search, mid-modal)
  need careful suppression.

### C. Both

Auto-watch as the default, with a manual reload binding kept
as the escape hatch (also useful for "I know something
changed but fsnotify missed it" cases on flaky filesystems).

- **Pro:** Best UX — live updates by default, explicit reload
  when needed.
- **Con:** All of B's cost plus the binding.

## iris's lean

**C, with B's complexity as the gating question.** Manual reload
is so cheap it should ship regardless — even if we never do
auto-watch, `r` to reload is ~20 lines of code and a test.
Auto-watch on top is the real conversation: is the UX win worth
the deps and the state-preservation work?

If the answer is "ship A first, decide on B later," that's a
fine path. A is a small follow-up to PR #13 cleanup; B is its
own feature card with a real design discussion.

## Open questions

1. How often does the TUI actually run while external mutations
   happen? Today, mostly never (single user, one terminal).
   With the MCP server in agent workflows, probably constantly.
2. State preservation on reload: cursor position, active
   filter, scroll offset, expanded card. Reload-by-rebuild
   loses all of this; we'd need to round-trip selection through
   card UUID, not row index.
3. Mid-modal behavior: if an external write lands while the
   user is typing in `:search`, do we redraw? My instinct:
   defer the reload until the modal closes. fsnotify gives us
   an event queue; we can drain it on modal close.

## Done when

- Decision made: A only / A + B / never.
- If A: build card filed for the manual reload binding (small).
- If B: build card filed with the design choices (debounce,
  state preservation, modal-deferred redraws) locked.

## Evidence

- bubbles v2 KeyMap landed in PR #13 — a `Reload` binding fits
  cleanly into the existing pattern.
- fsnotify: github.com/fsnotify/fsnotify (Go-idiomatic, used
  by hashicorp/consul, prometheus, cosmos-sdk; cross-platform
  with the caveats above).

## Out of scope

- Real-time collaboration features (presence, locks). The
  design doc is single-user.
- Server-side push (SSE, websockets). The TUI talks to the
  filesystem; that's the contract.

## Provenance

Filed 2026-05-05 by iris at Zach's request after the #0004
archival round.
