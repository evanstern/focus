---
schema_version: 2
id: 2
uuid: 019def84-2542-7dd7-911b-b56c61716edd
title: CLI shell completions (bash + zsh + fish)
type: card
status: backlog
priority: p2
project: focus
created: 2026-05-03
contract:
  - focus completions bash prints valid bash completion script
  - focus completions zsh prints valid zsh completion script
  - focus completions fish prints valid fish completion script
  - Subcommands complete (init, new, show, edit, activate, park, done, kill, revive, board, list, reindex, epic, mcp, tui, completions, version, help)
  - Card ids complete on commands that take an id (show, edit, activate, park, done, kill, revive)
  - Statuses complete on `focus list <TAB>` (active backlog done archived)
  - Flags complete (--priority, --type, --epic, --project, --slug, --force)
  - Priority values complete (p0 p1 p2 p3) on `--priority <TAB>`
  - Type values complete (card epic) on `--type <TAB>`
  - README install section documents how to load completions for each shell
  - "Tests cover at least: subcommand list parses, card-id producer returns the right ids"
---

## Brief

v1 had bash + zsh completions via a `focus completions` subcommand
that printed a hand-written script. v2 dropped them in the cutover.
Restore them — and add fish.

## Approach

Single approach: handwritten scripts, embedded with `//go:embed`.

We're not using cobra (decision in `wiki/decisions/focus-stack.md`),
so we don't get its auto-generated completion scripts for free. The
v1 scripts are the right shape — short, predictable, no
metaprogramming. Port them to v2's command surface.

```
internal/completions/
  bash.sh     # //go:embed source
  zsh.zsh     # //go:embed source
  fish.fish   # //go:embed source
  completions.go  # subcommand dispatcher: prints the embedded blob
```

`focus completions <shell>` prints the embedded script to stdout.
User pipes into `eval` (bash/zsh) or sources directly (fish).

## Dynamic completions (card ids, etc.)

Bash/zsh can shell out to `focus list --ids-only` (or similar) to
get the current card ids for completion. Add a quiet flag to
`focus list` or a dedicated `focus _complete <kind>` subcommand
that prints completion candidates by kind:

- `focus _complete ids` — all card ids
- `focus _complete ids --status active` — filtered ids
- `focus _complete priorities` — `p0 p1 p2 p3`
- `focus _complete types` — `card epic`
- `focus _complete statuses` — `active backlog done archived`
- `focus _complete subcommands` — hardcoded subcommand list

The `_complete` namespace is conventionally hidden from `focus
help` output. Underscore prefix flags it as internal.

## Why three shells

- bash: still the lowest common denominator on Linux
- zsh: macOS default + heavy among the audience
- fish: smaller share but loud users; cheap to add

Skip powershell/cmd. Out of scope.

## Install UX

README:

```bash
# bash — add to ~/.bashrc
eval "$(focus completions bash)"

# zsh — add to ~/.zshrc
eval "$(focus completions zsh)"

# fish — add to ~/.config/fish/completions/focus.fish
focus completions fish > ~/.config/fish/completions/focus.fish
```

## Tests

`internal/completions/completions_test.go`:

- Each shell's script parses without error in that shell (where the
  shell is available on the runner; gate behind build tags or skip
  if absent).
- `focus _complete ids` produces the expected ids on a tempdir
  board with a few cards in each status.
- `focus _complete priorities` is deterministic.

## Out of scope

- Powershell, cmd, nushell.
- Description text on completions (zsh's rich completion descriptions).
  Plain candidate completion is enough for v0.1.x.
- Caching completion candidates. Each tab call re-runs `focus
  _complete`; on a 1000-card board this is still sub-100ms because
  the index is the only thing read.

## Reference

v1's completion scripts are still reachable via the `v1-final` tag.
Not a port target — v2 has different commands and flags — but a
useful sanity check for the bash/zsh skeleton.
</content>
