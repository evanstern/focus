---
schema_version: 2
id: 8
uuid: 019dfa36-8e2c-7659-bbef-5da960c23e1b
title: 'Bug: focus kill rewrites unrelated cards'' frontmatter'
type: card
status: backlog
priority: p1
project: focus
created: 2026-05-05
tags: [bug, cli, data-integrity]
---

> **Merged into card #0006** for delivery (single feature
> session covers TUI cleanup + #0008 + #0009). This card stays
> in backlog as searchable history; it'll be archived when
> #0006 ships. The contract bullets are duplicated on #0006.

## Summary

`focus kill <id>` modified the frontmatter of an **unrelated**
card during a kill operation. Specifically, killing card #0004
flipped card #0002's `status` from `archived` back to `backlog`.

## Repro

On commit `888eb6f` (v0.1.3), with #0002 archived and #0004 in
backlog:

```
$ git status   # clean
$ focus kill 4 --force
#0004 → archived
$ git status
modified:   .focus/cards/0002-cli-shell-completions-bash-zsh-fish/INDEX.md
modified:   .focus/cards/0004-tui-discuss-charts-via-nimblemarkets-ntcharts/INDEX.md
```

The #0002 diff:

```
-status: archived
+status: backlog
```

Nothing else on #0002 changed.

## Why this matters

- **Data integrity:** a kill should be a one-card operation.
  Touching any other card during kill is a write-amplification bug.
- **Silent corruption:** I only caught it because I always
  inspect `git status` before committing. An agent that just runs
  `focus kill ... && git add . && git commit` will silently
  re-open archived cards.
- Already shipped this once — commit `7057ea5` on main carries
  the damage; fixed in `aea3f68` immediately after.

## Likely root cause (hypothesis)

`focus kill` probably rewrites every card's `INDEX.md` from the
in-memory index, and the in-memory state for #0002 was stale
(showing `backlog` from a previous unsaved transition). Either:

1. Save scope is too wide — kill is rewriting all cards, not
   just the target — or
2. The index is hydrated from somewhere lossy that doesn't read
   the on-disk frontmatter as source of truth.

Probably the same code path affects `done`, `park`, `activate`,
`revive` — the contract is that all of these should be
single-card mutations.

## Done when

- `focus kill <id>` only modifies
  `.focus/cards/<dir>/INDEX.md` for the targeted card and
  `.focus/index.json`.
- Same guarantee for `activate`, `park`, `done`, `revive`.
- Test: with N cards, run a kill and assert exactly two paths
  appear in `git status --porcelain` (the target's INDEX.md and
  the index.json).
- Test: any state-mutating command on card X must not change the
  on-disk content of any other card's INDEX.md.

## Out of scope

- Reformatting fixes for the body / tags round-trip — see card
  #0009. Those are related but distinct: #0008 is about *which*
  files get written, #0009 is about *how* they get written.

## Evidence

- Damage commit: `7057ea5`.
- Fix-forward: `aea3f68`.
- Hit during card #0004 archival on 2026-05-05.

## Provenance

Filed 2026-05-05 by iris. Caught during routine card archival.
