---
schema_version: 2
id: 9
uuid: 019dfa36-8e3b-7345-91c4-fd65791dea48
title: 'Bug: focus card save reformats body and tags (lossy markdown round-trip)'
type: card
status: backlog
priority: p1
project: focus
created: 2026-05-05
tags: [bug, cli, formatting]
---

> **Merged into card #0006** for delivery (single feature
> session covers TUI cleanup + #0008 + #0009). This card stays
> in backlog as searchable history; it'll be archived when
> #0006 ships. The contract bullets are duplicated on #0006.

## Summary

When focus mutates a card's frontmatter (e.g. via `kill`,
`activate`, `done`), the resulting INDEX.md is re-emitted with
different formatting than the input. Two observed effects on
card #0004 during a `focus kill 4`:

1. **Tags exploded from inline to block list:**

   ```
   -tags: [tui, design]
   +tags:
   +  - tui
   +  - design
   ```

2. **Body paragraphs unwrapped:** lines that were hard-wrapped
   at ~70 chars in the source got concatenated into single long
   lines. Visible in the same #0004 diff — every paragraph
   shrank in line-count and grew in line-length.

## Why this matters

- **Diff noise.** A status flip should be a one-line diff.
  Reformatting turns it into a 50-line diff that hides the real
  change.
- **Author intent erased.** Whoever wrote the card chose the
  line wrap and the inline-vs-block tags shape. Tools shouldn't
  silently rewrite them.
- **Round-trip lossiness blocks editor workflows.** `focus edit`
  → `:wq` → `focus done` will mangle the body even if you didn't
  touch it.

## Repro

Same as #0008 — observed in commit `7057ea5`. The body of
#0004's INDEX.md changed despite the operation only needing to
flip `status: backlog` → `status: archived`.

## Likely root cause (hypothesis)

The save path probably parses the markdown into an AST or struct,
then re-serializes from that representation. Round-tripping
markdown through any AST is famously lossy — list flow style,
paragraph wrapping, table alignment, and trailing whitespace are
all canonicalization decisions the serializer has to make, and
they almost never match the original.

The fix is to **never re-serialize the body**. Treat the body as
opaque bytes; only rewrite the frontmatter block.

## Done when

- Frontmatter mutations (`kill`, `done`, `activate`, `park`,
  `revive`) preserve the body byte-for-byte.
- Frontmatter mutations preserve the *unmodified* frontmatter
  fields byte-for-byte (only the changed key is rewritten).
- Specifically: tags written as `[a, b]` stay `[a, b]`; tags
  written as a block list stay a block list.
- Test: write a card with mixed-style frontmatter and a
  hand-wrapped body; flip its status; assert the diff is exactly
  one line.
- Test: round-trip `focus edit` (no edits, save) produces a
  byte-identical file.

## Out of scope

- The "wrong cards get touched" bug — see #0008.
- A pretty-printer / canonicalizer subcommand. If users want
  reformat, that's an explicit `focus fmt` call, not a side
  effect of state mutations.

## Implementation note

Recommend treating INDEX.md as a two-region file:

- **Frontmatter region:** parse + edit + re-serialize, but only
  the changed key is rewritten; all other keys preserved as bytes.
- **Body region:** pass through verbatim.

A YAML library that exposes the original token stream (rather
than parse-to-map-to-emit) is the right primitive for the
frontmatter region. If the current YAML library doesn't support
this, the cheapest fix is regex-based — find the `status:` line,
swap the value, leave everything else alone.

## Provenance

Filed 2026-05-05 by iris. Caught during routine card archival.
