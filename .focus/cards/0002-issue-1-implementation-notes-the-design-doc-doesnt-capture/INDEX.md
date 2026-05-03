---
schema_version: 2
id: 2
uuid: 019dec6b-f502-768c-8f76-5d3b5b57e82f
title: 'Issue #1: implementation notes the design doc doesnt capture'
type: card
status: backlog
priority: p2
project: focus
created: 2026-05-03
---

## Summary

This card mirrors `~/agents/iris/designs/focus-issue-001.md`. Once
this branch lands and the new repo `evanstern/focus` is created,
file these notes as Issue #1 against the GitHub repo.

The body below is a verbatim copy as of branch creation; the
authoritative version is the design-doc file.

---

# Issue #1 — Implementation notes the design doc doesn't capture

This document captures decisions that came out of conversation
between Iris (orchestrator), Zach, and Evan during the v2 design
phase, but that didn't make it into `designs/focus-v2.md` because
they're operational rather than architectural. They're the
"why is it like that?" answers a future contributor would
otherwise re-derive from scratch.

If anything below contradicts the design doc, the design doc
wins. If anything below contradicts the design doc *and* you
think the design doc is wrong, file a follow-up.

## Concurrency

### `next_id` allocation under flock

Every mutating CLI command acquires `.focus/.lock` via
`gofrs/flock` for the entire duration of the operation. That
includes:

- `focus new` (allocates id from `next_id`)
- `focus activate / park / done / kill / revive` (mutates
  status and rewrites the affected card + index)
- `focus epic add` (mutates a card's `epic:` field)
- `focus reindex` (rebuilds the entire index)
- The MCP equivalents of all of the above

Read-only commands do **not** take the lock. They read
`index.json` once. Atomic rename on the writer side guarantees
they see a consistent snapshot.

`flock` is **advisory**, not mandatory. If a non-focus process
writes to the cards/ directory while focus is reading, focus
won't notice. That's fine — humans hand-editing during a CLI
run is a self-inflicted wound; we don't defend against it.

### Stale lock files

If a focus process dies holding the lock, the kernel releases it
when the file descriptor closes. No PID-file dance, no manual
cleanup. The `.lock` file itself stays on disk forever; that's
expected and harmless.

### `next_id` regeneration semantics

`focus reindex` walks `cards/` and computes
`next_id = max(id) + 1` over every card present, regardless of
status. The previous index's `next_id` is preserved if it was
higher than this computed value, so:

- If card #47 was killed and the directory hand-deleted,
  `reindex` does not give #47 to the next new card.
- The high-water mark survives `reindex`, archive, kill, and
  any combination thereof.

The single failure mode: if someone deletes `index.json` AND
hand-deletes the highest-id card AND runs `reindex`, the id
gets reused. That's a "you broke the invariants three ways"
scenario; we don't defend against it.

## Slug rules at creation

`focus new "title"` derives the folder slug from the title.
Algorithm:

1. Lowercase.
2. Strip non-ASCII via NFKD normalize + ASCII-only filter.
3. Replace runs of non-alphanumeric with single hyphens.
4. Trim leading + trailing hyphens.
5. Truncate to 64 chars at a hyphen boundary if possible (no
   splitting words).
6. If result is empty, error: "title produced empty slug; pass
   `--slug <slug>` explicitly."

Collision handling: if `<padded-id>-<slug>/` would collide with
an existing directory, append `-2`, `-3` to the slug (NOT the
id — id is in the prefix and already disambiguates). In
practice this never collides because the id prefix is unique;
collision can only happen if a human creates a directory by
hand. If that happens, focus errors rather than silently
overwriting.

## Glamour width gotcha

When rendering the card body in the TUI's detail view via
`glamour`, the width passed to `glamour.NewTermRenderer` MUST
subtract the viewport's border width and horizontal padding.
Forgetting this causes the markdown to overflow the viewport
and look broken.

Cache the rendered output by card id; re-rendering on every
keystroke is noticeable on cards with code blocks or tables.

## UUIDv7 clock-skew handling

`uuid.NewV7()` returns an error when the system clock goes
backwards by more than 10 seconds (a defense against
duplicate UUIDs across reboots). Treat this as a fatal error
in `focus new`. Don't retry, don't fall back to v4.

## CLI/MCP shared logic placement

Both `internal/cli` and `internal/mcp` are **thin wrappers**.
All actual logic lives in `internal/board` (or its sub-
packages: `internal/board/card`, `internal/board/index`,
etc. — depends on how big things get).

If a helper exists in only one of the two surfaces, it should
probably move to `internal/board`. coda-lite did the opposite
(siblings with duplicated helpers); we explicitly do not.

## bubbletea v2 module path

The Charm bubbletea v2 module is currently published under the
`charm.land` domain in some references and `github.com/
charmbracelet` in others. Verify at scaffold time.

## `focus init` minimal state

`focus init` creates exactly:

```
.focus/
  config.yaml          # empty file
  cards/               # empty directory
```

Nothing else. No `index.json` (it gets created on first `focus
new`), no `.lock` (created on demand), no starter card, no
README, no `.gitignore` for `index.json` (that's a separate
choice — see below).

### `.gitignore` strategy

Recommendation: gitignore `index.json` and `.lock`. Document
that `focus reindex` is the first command you run on a fresh
clone if you want to use focus immediately. `focus board`
should auto-reindex if `index.json` is missing — that's a
quality-of-life detail, low cost.

## Status transition validation

Each transition has a "from" check enforced by the CLI:

| command | requires status | check on failure |
|---|---|---|
| `focus activate` | `backlog` | error "card already active" / "card is done/archived" |
| `focus park` | `active` | error "card is not active" |
| `focus done` | `active` | error "card is not active" + contract check |
| `focus kill` | any | always succeeds |
| `focus revive` | `archived` | error "card is not archived" |

`--force` overrides all of the above. Hand-edits to `status:`
in YAML bypass the check entirely; on next CLI write, the
index reflects the file (no auto-correction).

### Contract check on `focus done`

If the card has a non-empty `contract:` list, `focus done`
prints the contract and prompts `Confirm [y/N]:` unless
`--force` or stdout is non-tty (in which case it errors with
the contract printed). The TUI shows the contract as a checkbox
list before transitioning.

We do NOT track which contract items are checked — they're
purely a human-readable acceptance criterion. If finer
granularity is wanted later, that's a separate feature.

## TUI vim-binding edge cases

### `K` for kill

Capital-K because lowercase-k is movement (up). Documented in
the help pane.

### Search behavior

`/<query>` filters the active list to matching cards only,
preserving column structure. `n` and `N` jump to next/prev
match. `esc` clears the search and restores the full list.

Search matches against: title, slug (folder name), tags,
project, owner. NOT body text in v0.1.0 — that requires
loading every card's body, which defeats the speed win.

### Command-mode autocomplete

`:` followed by tab completes from the registered command list.
`:re<tab>` → `:reindex`. Implementation: small static map of
commands; tab cycles through prefix matches.
