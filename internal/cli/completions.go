package cli

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/evanstern/focus/internal/board"
	"github.com/evanstern/focus/internal/board/card"
	"github.com/evanstern/focus/internal/completions"
)

func runCompletions(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: focus completions <bash|zsh|fish>")
		return 2
	}
	script, ok := completions.Script(args[0])
	if !ok {
		fmt.Fprintf(stderr, "focus: unknown shell %q (supported: bash, zsh, fish)\n", args[0])
		return 2
	}
	fmt.Fprint(stdout, script)
	return 0
}

func runComplete(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: focus _complete <subcommands|ids|priorities|types|statuses>")
		return 2
	}
	kind, rest := args[0], args[1:]

	switch kind {
	case "subcommands":
		completions.PrintSubcommands(stdout)
		return 0
	case "priorities":
		completions.PrintPriorities(stdout)
		return 0
	case "types":
		completions.PrintTypes(stdout)
		return 0
	case "statuses":
		completions.PrintStatuses(stdout)
		return 0
	case "ids":
		return runCompleteIDs(rest, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "focus: unknown _complete kind %q\n", kind)
		return 2
	}
}

func runCompleteIDs(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("_complete ids", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	status := fs.String("status", "", "filter by status")
	cardType := fs.String("type", "", "filter by type")
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return 2
	}

	cwd, err := os.Getwd()
	if err != nil {
		return 0
	}
	b, err := board.Open(cwd)
	if err != nil {
		return 0
	}
	if err := completions.PrintIDs(stdout, b, completions.IDFilter{
		Status: card.Status(*status),
		Type:   card.Type(*cardType),
	}); err != nil {
		fmt.Fprintf(stderr, "focus: %v\n", err)
		return 1
	}
	return 0
}
