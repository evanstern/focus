# Feature: focus v2 — Go rewrite of the kanban tool

**Spawned by:** iris (orchestrator)
**Branch:** `feature/focus-v2`
**Worktree:** `/home/coda/projects/focus-focus-v2/`
**Target merge:** `feature/focus-v2 → v2 → main` (cutover plan)

---

## Brief

Build `focus` v2: a single Go binary that replaces the existing
bash implementation. The design is fully specified in
`~/agents/iris/designs/focus-v2.md` (read it first; it's the
contract).

This is **not** a port of v1. It's a ground-up rewrite. The
existing v1 source files are visible in this worktree because
the branch was cut from `main`. **Delete them all** in the
first commit — `bin/`, `coda-handler.sh`, `completions/`,
`hooks/`, `Makefile`, `plugin.json`, `test/`, and the existing
`README.md`. They have no place in v2 and stay reachable via
the `v1-final` tag.

## Authoritative reference docs

In `~/agents/iris/`:

- `designs/focus-v2.md` — **the design doc.** Canonical record
  of every locked decision. If you have a question and it's
  not answered here, the design doc almost certainly has it.
- `designs/focus-issue-001.md` — **micro-decisions doc.** All
  the operational calls that came out of conversation but
  didn't make the design doc: concurrency invariants, slug
  rules, glamour width gotcha, UUIDv7 clock-skew handling,
  status-transition validation, and more. Read this second.
- `designs/focus-readme-draft.md` — README to use as a starting
  point for the v2 README.md (replace v1's existing one).
- `designs/focus-deck/index.html` — visual reference. 9-slide
  HTML deck of the architecture. Optional read; useful when
  trying to remember "what was the principle ordering again?"
- `wiki/decisions/focus-stack.md` — locked dependency picks
  with rationale per pick.
- `wiki/decisions/focus-id-strategy.md` — id (int) + uuid (v7);
  slug is folder-only.
- `wiki/decisions/focus-global-home.md` — what `~/.focus/` is
  for.
- `wiki/decisions/focus-tui-keybinds.md` — vim-aware-first key
  map.

## Phasing

Build in this order. Each phase ends with a commit and (if
appropriate) a working binary at HEAD.

### Phase 0 — Repo prep

1. `git rm -r` all v1 files. Empty working tree.
2. Drop in the new `README.md` from `designs/focus-readme-draft.md`
   (under the orchestrator's iris config dir).
3. Add a permissive `MIT` LICENSE.
4. Add `.gitignore` covering `*.lock`, `.focus/index.json`
   (we don't commit derived caches), `dist/`, OS detritus.
5. `go mod init github.com/evanstern/focus`.
6. Set Go version pin to a real version (NOT `1.25.0` — that
   doesn't exist as of 2026; coda-lite has this typo, don't
   propagate it).
7. Commit: "Phase 0: clear v1 source, scaffold v2 module skeleton."

### Phase 1 — `internal/board` core

The shared logic layer. Both CLI and MCP wrap this. **Do not
duplicate logic across surfaces** — that's an explicit
anti-pattern observed in coda-lite.

Sub-packages (rough; refactor as it grows):

- `internal/board` — top-level types, board resolution
  (ancestor walk for `.focus/`), config.
- `internal/board/card` — Card struct, frontmatter parse +
  re-marshal preserving unknown fields, validation.
- `internal/board/index` — index.json schema, atomic write
  via `google/renameio/v2`, `next_id` invariants.
- `internal/board/lock` — flock wrapper around
  `gofrs/flock`, scoped to mutating ops.

Required tests at minimum: card parse/marshal round-trip
including unknown-field preservation, next_id allocation under
concurrent writes (simulated), atomic-rename safety.

Commit after each sub-package is functional + tested.

### Phase 2 — CLI

Handwritten dispatch (NO cobra). Pattern per
`~/agents/iris/wiki/decisions/focus-stack.md`:

- `cmd/focus/main.go` is thin: parse args, call `cli.Run`,
  print error.
- `cli.Run(args, stdout, stderr io.Writer) int` — entire CLI
  callable from `go test` without exec'ing.
- `internal/cli/dispatch.go` — switch tree.
- One file per noun: `internal/cli/{init,new,show,edit,
  activate,park,done,kill,revive,board,list,reindex,epic}.go`
- Each handler is a thin wrapper: parse flags via
  `flag.NewFlagSet`, call `internal/board`, format output.

Acceptance: `focus init` + `focus new` + `focus show` + `focus
board` + `focus list` + every transition command + `focus
reindex` all work end-to-end against a tempdir board. Tests
cover the happy path of each.

Commit after each command works.

### Phase 3 — Dogfood checkpoint

Once Phase 2 is done:

1. `cd ~/projects/focus-focus-v2 && ./focus init`
2. `./focus new "Implement focus v2"` — files card #0001
3. `./focus new "Issue #1: implementation notes"` — card #0002
   (paste body from `designs/focus-issue-001.md`)
4. File any in-flight TODO items as cards.
5. From here on, track work with focus itself. The TUI doesn't
   exist yet but `focus board` and `focus list` work.

This is the eat-your-own-dogfood gate. If the CLI is too clunky
to actually use for tracking work at this point, fix the
clunkiness before moving on.

### Phase 4 — MCP server

`focus mcp serve` runs the official MCP SDK over stdio. Tools
are thin wrappers over `internal/board`. See
`designs/focus-issue-001.md` § "CLI/MCP shared logic placement"
for the pattern.

Wire it into iris's `opencode.json` as a local MCP server, test
that the orchestrator can read the board through it. From this
point I (iris) can manage focus cards via MCP tools.

Commit per tool implemented.

### Phase 5 — TUI

Bubble Tea + bubbles + lipgloss + glamour, vim-aware first.
See `wiki/decisions/focus-tui-keybinds.md` for the full key
map.

Architecture: state enum + delegated updates. Reference
implementations to crib from:
- `charmbracelet/kancli`
- `bborn/workflow`
- `happytaoer/cli_kanban`

Glamour width gotcha (subtract border + padding) is in Issue
#1.

Commit per major view (board → detail → search → command-mode).

### Phase 6 — Release prep

1. `goreleaser` config (`.goreleaser.yaml`) targeting linux +
   darwin × amd64 + arm64.
2. GitHub Actions workflow to build + release on tag push.
3. README polish: real install instructions with current SHA.
4. Tag `v0.1.0`, push.
5. Manual: cut a GitHub release with the goreleaser artifacts.

### Phase 7 — Cutover

1. Open PR `feature/focus-v2 → v2`.
2. Once merged, open PR `v2 → main`.
3. Once merged, this is now mainline focus.

## Done when

- [ ] All v1 source removed; `go.mod` initialized
- [ ] `internal/board` core complete + tested
- [ ] CLI implements all design-doc commands
- [ ] Dogfood checkpoint passed (this worktree's `.focus/`
      tracks the remaining work)
- [ ] MCP server runs; iris can manage cards through it
- [ ] TUI lands with vim keybinds
- [ ] `v0.1.0` tagged with goreleaser binaries
- [ ] `feature/focus-v2 → v2 → main` merged
- [ ] Issue #1 ("Implementation notes the design doc doesn't
      capture") filed against the new repo

## Notes

- **Worktree structure.** This is a worktree off
  `~/projects/focus/.bare/`. Don't `git clone` again or operate
  outside this dir.
- **Design doc is canonical.** If a decision in this brief
  contradicts the design doc, the design doc wins. Update the
  brief.
- **Ask iris in inbox** if you hit ambiguity or want to push
  back on a design decision. `coda-lite msg iris "..."`.
- **Memory + wiki.** Write to your own session memory as you
  go; promote durable patterns to iris's wiki via inbox.
</content>
