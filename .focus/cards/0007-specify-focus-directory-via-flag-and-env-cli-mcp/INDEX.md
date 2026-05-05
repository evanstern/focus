---
schema_version: 2
id: 7
uuid: 019dfa36-8e1b-7d74-8545-2e01ab85fcde
title: Specify focus directory via flag and env (CLI + MCP)
type: card
status: backlog
priority: p1
project: focus
created: 2026-05-05
tags: [cli, mcp, dx]
---

## Summary

Make focus operate against an explicit `.focus/` location instead
of always discovering from `$PWD` upward. Both the CLI and the
MCP server need this.

## Why now

The MCP server was rooted on a `.focus/` board different from the
focus repo I wanted to act on (it grabbed `the-stacks/.focus/`
instead of `focus/main/.focus/`). I had to drop down to
`cd /home/coda/projects/focus/main && focus kill 4` to actually
archive the right card. Same class of friction will hit agents,
scripts, and anyone running focus from outside their project tree.

## Concrete shape

### CLI

Two surfaces, flag wins over env, env wins over walk:

- `--focus-dir <path>` — global flag accepted by every subcommand.
  Points at either a `.focus/` directory directly or at a project
  root that contains one.
- `FOCUS_DIR` env var — same semantics.

Resolution order:

1. `--focus-dir <path>` if set
2. `$FOCUS_DIR` if set
3. Walk upward from `$PWD` (current behavior)

If a path is given that doesn't contain `.focus/` (or isn't itself
a `.focus/`), error out clearly: `focus: no .focus/ found at <path>`.

### MCP

Mirror the CLI. The MCP server should accept either:

- A startup arg / env (e.g. `FOCUS_DIR=...` in the MCP launch
  config) that pins the server to one board, **or**
- A per-tool optional `focus_dir` argument so a single MCP server
  can act across multiple boards.

Pick one. Recommendation in the brief: **per-tool argument**,
optional, falls back to server default (env or upward walk). Reason:
a single agent often manages multiple projects, and respawning the
MCP server per project is friction.

## Done when

- `focus --focus-dir <path> <subcommand>` works for every
  subcommand.
- `FOCUS_DIR=<path> focus <subcommand>` works for every subcommand.
- Resolution precedence is documented in `focus --help` and README.
- MCP tools accept an optional `focus_dir` argument; behavior
  documented in the MCP server's tool schemas.
- Tests cover CLI flag + env precedence (table-driven), error
  message when path has no `.focus/`, and MCP per-tool argument
  round-tripping through to the same resolver.

## Out of scope

- Changing the upward-walk default behavior.
- Multi-board operations in a single command (e.g.
  `focus list --all-boards`).

## Provenance

Filed 2026-05-05 by iris after the MCP-vs-CLI board mismatch
during card #0004 archival. Same class of friction will hit any
agent or script not running from inside the project worktree.
