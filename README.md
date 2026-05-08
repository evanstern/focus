# focus

A project-local kanban board for solo developers and their
orchestrator agents. One Go binary. No daemon, no database, no
sync. The filesystem is the source of truth.

```bash
$ cd ~/projects/myapp
$ focus init
$ focus new "Wire up the auth service"
0001
$ focus board
ACTIVE (0/3)
  (none)

BACKLOG
  #0001  Wire up the auth service                  myapp       p2    -
$ focus activate 1
$ focus done 1
```

## What it is

`focus` walks up from `$PWD` looking for a `.focus/` directory,
the way `git` walks for `.git/`. Found one? You're on its board.
Cards are markdown files with YAML frontmatter, organized one
folder per card so designs and screenshots can live next to the
card itself.

Three surfaces, one tool:

- **CLI** — `focus new`, `focus done`, `focus board`, etc.
- **TUI** — `focus tui` opens a vim-keybind interactive board.
- **MCP** — `focus mcp serve` exposes the same operations as
  Model Context Protocol tools, for orchestrator agents.

Four statuses (`active`, `backlog`, `done`, `archived`), no more.

## Install

### Go install (always-current)

```bash
go install github.com/evanstern/focus/cmd/focus@latest
```

### Pre-built binaries

Download from [GitHub Releases](https://github.com/evanstern/focus/releases).
linux/amd64, linux/arm64, darwin/amd64, darwin/arm64.

```bash
# example for linux/amd64; replace VERSION with the tag you want
VERSION=v0.1.0
curl -L "https://github.com/evanstern/focus/releases/download/${VERSION}/focus-${VERSION#v}-linux-amd64.tar.gz" \
  | tar xz
sudo mv focus /usr/local/bin/
```

## Shell completions

`focus completions <shell>` prints a completion script to stdout.
Subcommands, card ids, statuses, priorities, and types complete on
`<TAB>`. Card-id completions are filtered by command: `focus done <TAB>`
only offers active cards, `focus revive <TAB>` only archived ones.

```bash
# bash — add to ~/.bashrc
eval "$(focus completions bash)"

# zsh — add to ~/.zshrc (after compinit)
eval "$(focus completions zsh)"

# fish
focus completions fish > ~/.config/fish/completions/focus.fish
```

## Quickstart

```bash
# 1. Create a board in your project
cd ~/projects/myapp
focus init

# 2. File a card
focus new "Wire up the auth service" --priority p1

# 3. List cards
focus board                  # active + backlog
focus list done              # filter by status

# 4. Move cards through the lifecycle
focus activate 1             # backlog → active
focus done 1                 # active → done
focus kill 5                 # any → archived

# 5. Browse interactively
focus tui

# 6. Hand to an orchestrator
focus mcp serve              # JSON-RPC over stdio
```

## Card format

Each card lives in `.focus/cards/<padded-id>-<slug>/INDEX.md`:

```markdown
---
schema_version: 2
id: 142
uuid: 7f3a9b2c-9e1d-4f8a-b5e1-6e2d8f1a3c4b
title: Ship the feature
type: card
status: backlog
priority: p2
project: api
created: 2026-05-04
---

## Summary

Free-form markdown body.
```

The folder slug is human navigation aid only — it's not in the
frontmatter. Cards are referenced by `id` within a board. The `uuid`
is reserved for future cross-board features but is not yet a lookup
key in v0.1.0.

## Configuration

`focus` resolves which board to operate on in this order:

1. `--focus-dir <path>` — root flag, available on every subcommand.
2. `FOCUS_DIR` env var — same effect, useful for shells and CI.
3. Upward walk from `$PWD` looking for `.focus/`, the way `git`
   walks for `.git/`.

`<path>` may be either a project root (the directory that
contains `.focus/`) or the `.focus/` directory itself.

```bash
# operate on a board you're not currently inside
focus --focus-dir ~/projects/myapp board

# pin a shell to a specific board
export FOCUS_DIR=~/projects/myapp
focus list
```

If the supplied path doesn't resolve to a `.focus/`, focus exits
with `focus: no .focus/ found at <path>`.

The MCP server uses the same chain at startup to pick its default
board (env + upward walk from the server's CWD). Each MCP tool
also accepts an optional `focus_dir` argument that overrides the
default for a single call — handy when one long-running server is
juggling multiple boards.

## Architecture overview

```
~/projects/myapp/
  .focus/
    config.yaml
    index.json          # derived cache
    cards/
      0001-add-mcp/
        INDEX.md
        design.md       # optional artifact, lives next to card
      0002-fix-auth/
        INDEX.md
```

Plus a global `~/.focus/` for cross-board state (config,
intent, migration orphans). No global card store; boards stay
project-local.

## Status

**v0.1.0 in development.** v1 (the original bash implementation)
is reachable via the `v1-final` tag. v2 is a ground-up Go
rewrite; it does not migrate v1 cards.

## License

MIT. See [LICENSE](LICENSE).

## Author

Evan Stern, with substantial design + build contributions from
[iris](https://github.com/evanstern/) (LLM orchestrator agent).
