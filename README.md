# focus

A kanban card manager for personal task tracking. Cards are markdown files with YAML frontmatter. `focus` is the only interface for state transitions.

## Install

```sh
git clone <this-repo> && cd focus
make install          # installs to /usr/local/bin
# or
PREFIX=~/.local make install
```

Add tab completions to your shell:

```sh
# bash — add to ~/.bashrc
eval "$(focus completions bash)"

# zsh — add to ~/.zshrc
eval "$(focus completions zsh)"
```

## Quick start

```sh
focus init                        # create a new kanban board
focus new "Ship the feature" web  # create a card
focus activate 1                  # start working on it
focus board                       # see your board
focus done 1                      # mark it done
```

## Commands

| Command | Description |
|---|---|
| `board` | Show active + backlog cards |
| `new "title" [project]` | Create a backlog card (add `--type milestone` for a milestone) |
| `milestone <id>` | Show milestone detail with progress + children |
| `milestone new "title" [project]` | Shortcut for creating a milestone card |
| `milestone list` | List milestones with progress summaries |
| `milestone add <mid> <cid>` | Link a card to a milestone |
| `show <id\|slug>` | Show card details |
| `activate <id\|slug>` | Move to active (enforces WIP limit) |
| `park <id\|slug>` | Move to parked |
| `kill <id\|slug>` | Move to killed |
| `done <id\|slug>` | Move to done (checks completion contract) |
| `edit <id\|slug>` | Open in `$EDITOR` |
| `intent [message]` | Set/show session intent (tmux-aware) |
| `wip` | Show WIP status |
| `list [status]` | List cards, optionally filtered |
| `init [dir]` | Create a new kanban board (default: `~/.focus/kanban`) |
| `setup <path>` | Point focus at an existing kanban directory |
| `tui` | Interactive terminal UI (navigate, search, transition cards) |
| `completions [bash\|zsh]` | Print shell completions for eval |

## Flags

- `--force` — Bypass WIP limit and contract checks
- `--quiet` — Suppress non-essential output
- `--project <name>` — Filter by project
- `--priority <p0-p3>` — Filter by priority
- `--no-color` — Disable color (or set `NO_COLOR`)

## Card format

Cards are markdown files in the kanban directory:

```markdown
---
id: 1
title: Ship the feature
project: web
status: backlog
priority: p2
created: 2025-01-15
updated: 2025-01-15
contract:
  - Tests pass
  - Code reviewed
---

## Notes

Implementation details go here.
```

## Milestones

A milestone is a card with `type: milestone`. Regular cards link up via a
`milestone: <id>` frontmatter field — one card, one milestone, no nesting.
Milestones support cross-project work: children can have any `project:` value.

```sh
focus milestone new "Launch v2" web        # create a milestone
focus new "Ship backend" api               # create a regular card
focus milestone add 1 2                    # link card #2 to milestone #1
focus milestone 1                          # show progress and children
focus milestone list                       # summary of all milestones
```

`focus done <milestone-id>` blocks if any child is still `active` or
`backlog`; use `--force` to override. `focus board` shows an additional
`MILESTONES` section grouping active/backlog children under their milestone.

## Configuration

All state lives under `~/.focus/` by default:

```
~/.focus/
  config          # YAML config file
  kanban/         # card files (default board location)
  intents/        # session intent files
```

The config file (`~/.focus/config`) is YAML:

```yaml
kanban_dir: /path/to/your/kanban
wip_limit: 3
```

Environment variables override config file values:

| Variable | Default | Description |
|---|---|---|
| `FOCUS_HOME` | `~/.focus` | Base directory for all focus data |
| `FOCUS_KANBAN_DIR` | `~/.focus/kanban` | Card directory |
| `FOCUS_WIP_LIMIT` | `3` | Max active cards |

## Coda plugin

Focus integrates with [coda](https://github.com/evanstern/coda) as a plugin via coda's plugin system. This adds:

- **`coda focus <command>`** — all focus commands accessible as a coda subcommand
- **MCP tools** — OpenCode agents can manage kanban cards (board, new, show, activate, done, park, kill, list, wip, intent)
- **Lifecycle hooks** — auto-sets focus intent when a feature branch is created, clears it on teardown

### Install

```sh
coda plugin install <this-repo-url>
```

Or if working from a local clone:

```sh
mkdir -p ~/.config/coda/plugins
ln -s /path/to/focus ~/.config/coda/plugins/focus
```

Then add to `~/.config/coda/config.json`:

```json
{
  "plugins": {
    "<this-repo-url>": { "enabled": true }
  }
}
```

### Usage via coda

```sh
coda focus                # show the board
coda focus new "Fix bug"  # create a card
coda focus activate 1     # start working
coda focus done 1         # mark done
```

All focus commands and flags pass through.

### Uninstall

```sh
coda plugin remove focus
```

## Tests

```sh
make test    # requires bats
make lint    # requires shellcheck
```
