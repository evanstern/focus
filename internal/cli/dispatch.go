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
	"strings"

	"github.com/evanstern/focus/internal/board"
)

// Version is the focus binary version. Stamped at build time via
// goreleaser; defaults to "dev" for source builds.
var Version = "dev"

// focusDirFlag holds the value of --focus-dir extracted at dispatch
// time. Consulted by openBoard. Reset on each Run() entry so tests
// remain hermetic.
var focusDirFlag string

// Run executes the focus CLI. It returns the process exit code.
//
// Exit codes:
//
//	0 — success
//	1 — runtime error (board op failed, file IO, etc.)
//	2 — usage error (unknown command, missing args, bad flag)
func Run(args []string, stdout, stderr io.Writer) int {
	focusDirFlag = ""
	args, err := extractFocusDir(args)
	if err != nil {
		fmt.Fprintf(stderr, "focus: %v\n", err)
		return 2
	}
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
	case "completions":
		return runCompletions(rest, stdout, stderr)
	case "_complete":
		return runComplete(rest, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "focus: unknown command %q. try `focus help`.\n", cmd)
		return 2
	}
}

// openBoard resolves the focus board using the three-tier order
// flag > env > upward walk from $PWD. Handlers call this and bail
// with exit code 1 if resolution fails.
func openBoard(stderr io.Writer) (*board.Board, int) {
	dir, err := board.Resolve(focusDirFlag, os.Getenv("FOCUS_DIR"))
	if err != nil {
		var fnf *board.FocusDirNotFoundError
		switch {
		case errors.Is(err, board.ErrNotInBoard):
			fmt.Fprintln(stderr, "focus: not in a focus board. run `focus init` to create one here.")
		case errors.As(err, &fnf):
			fmt.Fprintln(stderr, fnf.Error())
		default:
			fmt.Fprintf(stderr, "focus: %v\n", err)
		}
		return nil, 1
	}
	b, err := board.OpenAt(dir)
	if err != nil {
		fmt.Fprintf(stderr, "focus: %v\n", err)
		return nil, 1
	}
	return b, 0
}

// extractFocusDir pulls --focus-dir / -focus-dir (with either =VALUE
// or a separate VALUE token) out of args before subcommand dispatch.
// This makes the flag work as a "persistent" root flag without
// requiring every subcommand's flagset to register it.
func extractFocusDir(args []string) ([]string, error) {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			out = append(out, args[i:]...)
			return out, nil
		}
		switch {
		case a == "--focus-dir" || a == "-focus-dir":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("flag needs an argument: %s", a)
			}
			focusDirFlag = args[i+1]
			i++
		case strings.HasPrefix(a, "--focus-dir="):
			focusDirFlag = strings.TrimPrefix(a, "--focus-dir=")
		case strings.HasPrefix(a, "-focus-dir="):
			focusDirFlag = strings.TrimPrefix(a, "-focus-dir=")
		default:
			out = append(out, a)
		}
	}
	return out, nil
}

const helpText = `focus — project-local kanban for developers and agents

USAGE
  focus [--focus-dir <path>] <command> [args]

CONFIGURATION
  Board location is resolved in this order (first match wins):
    1. --focus-dir <path>     persistent root flag
    2. FOCUS_DIR env var
    3. upward walk from $PWD looking for .focus/
  <path> may be a project root containing .focus/ or the .focus/
  directory itself.

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

SHELL COMPLETIONS
  completions <shell>      Print bash|zsh|fish completion script.

META
  version                  Print version
  help                     This message

See https://github.com/evanstern/focus for documentation.`
