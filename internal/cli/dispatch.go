// Package cli implements the focus command-line interface.
//
// Run is the in-process entry point: it accepts argv (without the
// program name), an stdout writer, and an stderr writer, and returns
// the process exit code. main.go is a thin shim that calls Run with
// os.Args[1:] and os.Stdout/os.Stderr.
//
// This shape lets the entire CLI be exercised from go test without
// exec'ing a subprocess. Each handler in this package follows the
// pattern: parse flags, call internal/board, format output. No
// business logic lives here — that's all in internal/board.
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/evanstern/focus/internal/board"
)

// Version is the focus binary version. Stamped at build time via
// goreleaser; defaults to "dev" for source builds.
var Version = "dev"

// Run executes the focus CLI. It returns the process exit code.
//
// Exit codes:
//
//	0 — success
//	1 — runtime error (board op failed, file IO, etc.)
//	2 — usage error (unknown command, missing args, bad flag)
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, helpText)
		return 2
	}

	cmd, rest := args[0], args[1:]
	switch cmd {
	case "version", "--version", "-v":
		fmt.Fprintln(stdout, Version)
		return 0
	case "help", "--help", "-h":
		fmt.Fprintln(stdout, helpText)
		return 0
	case "init":
		return runInit(rest, stdout, stderr)
	case "new":
		return runNew(rest, stdout, stderr)
	case "show":
		return runShow(rest, stdout, stderr)
	case "edit":
		return runEdit(rest, stdout, stderr)
	case "board":
		return runBoard(rest, stdout, stderr)
	case "list":
		return runList(rest, stdout, stderr)
	case "activate":
		return runActivate(rest, stdout, stderr)
	case "park":
		return runPark(rest, stdout, stderr)
	case "done":
		return runDone(rest, stdout, stderr)
	case "kill":
		return runKill(rest, stdout, stderr)
	case "revive":
		return runRevive(rest, stdout, stderr)
	case "reindex":
		return runReindex(rest, stdout, stderr)
	case "epic":
		return runEpic(rest, stdout, stderr)
	case "mcp":
		return runMCP(rest, stdout, stderr)
	case "tui":
		return runTUI(rest, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "focus: unknown command %q. try `focus help`.\n", cmd)
		return 2
	}
}

// openBoard resolves the nearest .focus/ from $PWD. Handlers that
// need a board call this and bail with exit code 1 if the resolution
// fails. For ErrNotInBoard we print the friendly error message that
// the design doc specifies.
func openBoard(stderr io.Writer) (*board.Board, int) {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "focus: %v\n", err)
		return nil, 1
	}
	b, err := board.Open(cwd)
	if err != nil {
		if errors.Is(err, board.ErrNotInBoard) {
			fmt.Fprintln(stderr, "focus: not in a focus board. run `focus init` to create one here.")
		} else {
			fmt.Fprintf(stderr, "focus: %v\n", err)
		}
		return nil, 1
	}
	return b, 0
}

const helpText = `focus — project-local kanban for developers and agents

USAGE
  focus <command> [args]

BOARD
  init [path]              Create a .focus/ at path (default: $PWD).
  reindex                  Rebuild .focus/index.json from cards/.
  board                    Show active + backlog (default view).
  list [status]            Flat list, filterable by --project,
                           --priority, --epic, --owner, --tag, --type.

CARDS
  new <title> [flags]      Create a new card.
                             --project <p>  --priority p0|p1|p2|p3
                             --epic <id>    --type card|epic
                             --slug <s>
  show <id>                Render card detail (frontmatter + body).
  edit <id>                Open INDEX.md in $EDITOR.

LIFECYCLE
  activate <id> [--force]  backlog → active (WIP-checked)
  park <id>                active → backlog
  done <id> [--force]      active → done (contract-checked)
  kill <id>                any → archived
  revive <id>              archived → backlog

EPICS
  epic <id>                Detail + progress.
  epic list                Summary of all epics in this board.
  epic add <epic-id> <card-id>
                           Set epic: on a card.

TUI
  tui                      Open the interactive board (vim keybinds).

MCP
  mcp serve                JSON-RPC over stdio for MCP clients.

META
  version                  Print version
  help                     This message

See https://github.com/evanstern/focus for documentation.`
