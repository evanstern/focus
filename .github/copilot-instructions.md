# Copilot Cloud Agent Instructions — focus

## Repository overview

`focus` is a personal kanban card manager written entirely in **Bash**. Cards are Markdown files with YAML frontmatter. `bin/focus` is the single entry-point script; it is the **only** interface for state transitions — never hand-edit `status:` fields directly.

The repository also ships as a **coda plugin** (`plugin.json`, `coda-handler.sh`) that exposes focus commands and MCP tools to the [coda](https://github.com/evanstern/coda) CLI agent framework.

## Repository layout

```
bin/focus               Main CLI script (~64 KB, pure Bash)
coda-handler.sh         Coda plugin dispatcher (sourced by coda, not executed directly)
plugin.json             Coda plugin manifest (commands, MCP tools, hooks)
Makefile                install / uninstall / test / lint targets
completions/
  focus.bash            Bash tab-completion script
  _focus                Zsh tab-completion script
hooks/
  post-feature-create/50-focus-intent       Sets session intent on branch creation
  pre-feature-teardown/50-focus-clear-intent  Clears intent on teardown
test/
  focus.bats            bats test suite
  fixtures/             Sample card .md files used by tests
```

## Language and tooling

- **Language**: Bash only — no other runtime is required.
- **Test framework**: [bats](https://github.com/bats-core/bats-core) (`bats test/`).
- **Linter**: [shellcheck](https://www.shellcheck.net/) (`shellcheck bin/focus completions/focus.bash`).
- **Build/install**: GNU Make (`make install`, `make test`, `make lint`).

### Run tests and linting

```sh
make test    # requires bats to be on PATH
make lint    # requires shellcheck to be on PATH
```

To install bats and shellcheck in the agent's environment:

```sh
apt-get install -y bats shellcheck
```

Tests use an isolated `$BATS_TEST_TMPDIR` as `FOCUS_HOME` — no real `~/.focus` directory is touched.

## Card format

Every card is a `.md` file in `$FOCUS_KANBAN_DIR` (default `~/.focus/kanban`):

```markdown
---
id: 1
title: Ship the feature
project: web
status: backlog          # backlog | active | done | parked | killed
priority: p2             # p0 (highest) … p3 (lowest)
created: 2025-01-15
updated: 2025-01-15
type: milestone          # only present on milestone cards
milestone: 1             # only present when card is linked to a milestone
contract:
  - Tests pass
  - Code reviewed
---

## Notes

Free-form markdown body.
```

- Card filenames are the slugified title (e.g. `ship-the-feature.md`).
- The `id` field is a monotonically increasing integer assigned at creation time.
- Cards can be referenced by numeric ID or by slug on the command line.
- **Never directly edit** `status:`, `id:`, `updated:`, or `milestone:` fields — always use the `focus` CLI.

## Key commands

```sh
focus init                        # create a new kanban board
focus new "Title" [project]       # create a backlog card
focus activate <id|slug>          # move to active (respects WIP limit)
focus done <id|slug>              # mark done (checks contract)
focus park <id|slug>              # move to parked
focus kill <id|slug>              # move to killed
focus board                       # show active + backlog
focus list [status]               # list all or filtered cards
focus show <id|slug>              # card detail
focus milestone new "Title" [project]   # create milestone card
focus milestone add <mid> <cid>   # link card to milestone
focus milestone <id|slug>         # show milestone with progress
focus milestone list              # list all milestones
focus intent [message]            # set/show session intent
focus wip                         # WIP status
focus tui                         # interactive TUI
```

Useful flags: `--force` (bypass WIP limit / contract checks), `--quiet`, `--no-color`, `--project <name>`, `--priority <p0-p3>`.

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `FOCUS_HOME` | `~/.focus` | Base directory |
| `FOCUS_KANBAN_DIR` | `~/.focus/kanban` | Card directory |
| `FOCUS_WIP_LIMIT` | `3` | Max active cards |
| `NO_COLOR` | — | Set to any value to disable color |

Tests set `FOCUS_HOME`, `FOCUS_KANBAN_DIR`, `FOCUS_INTENT_DIR`, and `NO_COLOR` to isolated temp dirs.

## Coding conventions

- `set -euo pipefail` is active throughout `bin/focus`.
- Internal helper functions are prefixed `_focus_`.
- Color output goes through `_color()` which respects `$COLOR_ENABLED` / `$NO_COLOR`.
- Field reads from frontmatter use `_focus_read_field <file> <field>`.
- Field writes use `_focus_set_field <file> <field> <value>`.
- Card lookup accepts both numeric IDs and slugs via `_focus_card_path`.
- Sorting uses `_focus_sorted_files_for_status` (sort by priority rank, then updated desc, then id).
- All commands exit non-zero on errors and print to stderr; normal output goes to stdout.

## WIP limit and completion contract

- `focus activate` blocks when active card count ≥ `$FOCUS_WIP_LIMIT`; use `--force` to override.
- `focus done` blocks if a `contract:` list exists and any item is not checked off in the body (`- [x]`); use `--force` to override.
- `focus done <milestone>` blocks if any child card is still `active` or `backlog`; use `--force` to override.

## Coda plugin

- `plugin.json` declares commands, MCP tools, shell completions, and lifecycle hooks.
- `coda-handler.sh` is **sourced** (not executed) by coda's plugin dispatcher; functions named `_coda_focus_*` are called with `--key value` argument style.
- Hooks in `hooks/` are executable scripts called by coda on feature-branch lifecycle events.

## Testing approach

- All tests are in `test/focus.bats` using bats syntax (`@test "description" { … }`).
- Fixtures in `test/fixtures/` are copied into `$FOCUS_KANBAN_DIR` via `load_fixture`.
- Tests rely on the `$FOCUS` variable pointing at `bin/focus` and use `run` to capture output and exit status.
- When adding new commands or modifying existing behavior, add or update `@test` blocks in `focus.bats`.

## Known workarounds / environment notes

- shellcheck may warn about `SC2207` (array from command substitution) — existing code uses `while IFS= read -r` patterns for safe iteration instead.
- The `sed -i` usage is GNU-style (no backup extension); macOS `sed` requires `sed -i ''` — keep this in mind if the agent environment is non-Linux.
- `focus tui` requires a real terminal (uses ANSI escape sequences and reads keyboard input) — do not call it in non-interactive CI contexts.
