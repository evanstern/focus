// Package cli implements the focus command-line interface.
//
// Run is the in-process entry point: it accepts argv (without the
// program name), an stdout writer, and an stderr writer, and returns
// the process exit code. main.go is a thin shim that calls Run with
// os.Args[1:] and os.Stdout/os.Stderr.
//
// This shape lets the entire CLI be exercised from go test without
// exec'ing a subprocess.
package cli

import (
	"fmt"
	"io"
)

// Version is the focus binary version. Stamped at build time via
// goreleaser; defaults to "dev" for source builds.
var Version = "dev"

// Run executes the focus CLI. It returns the process exit code.
//
// Phase 0 stub: only `focus version` and `focus help` are wired up.
// Subsequent phases fill in the dispatch tree.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "focus: no command given. try `focus help`.")
		return 2
	}

	switch args[0] {
	case "version", "--version", "-v":
		fmt.Fprintln(stdout, Version)
		return 0
	case "help", "--help", "-h":
		fmt.Fprintln(stdout, helpText)
		return 0
	default:
		fmt.Fprintf(stderr, "focus: unknown command %q. try `focus help`.\n", args[0])
		return 2
	}
}

const helpText = `focus — project-local kanban for developers and agents

USAGE
  focus <command> [args]

COMMANDS
  (Phase 0 scaffold — most commands not yet implemented.)

  version          Print version and exit
  help             Show this message

See https://github.com/evanstern/focus for documentation.`
