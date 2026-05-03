package cli

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/evanstern/focus/internal/board"
	"github.com/evanstern/focus/internal/board/card"
)

// transitionHandler is the shape of every status-transition CLI
// handler: it parses (id [--force]), opens the board, and calls the
// supplied op. Same boilerplate for activate / park / kill / revive,
// so we factor it. Done has its own handler because of the contract
// prompt.
type transitionFn func(b *board.Board, id int, force bool) (*card.Card, error)

func runTransition(name string, op transitionFn, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	force := fs.Bool("force", false, "skip from-status validation")
	fs.Usage = func() { fmt.Fprintf(stderr, "usage: focus %s <id> [--force]\n", name) }
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fs.Usage()
		return 2
	}
	id, err := strconv.Atoi(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(stderr, "focus: invalid id %q\n", fs.Arg(0))
		return 2
	}

	b, code := openBoard(stderr)
	if code != 0 {
		return code
	}
	c, err := op(b, id, *force)
	if err != nil {
		fmt.Fprintf(stderr, "focus: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "#%s → %s\n", card.PaddedID(c.ID), c.Status)
	return 0
}

func runActivate(args []string, stdout, stderr io.Writer) int {
	return runTransition("activate", (*board.Board).Activate, args, stdout, stderr)
}

func runPark(args []string, stdout, stderr io.Writer) int {
	return runTransition("park", (*board.Board).Park, args, stdout, stderr)
}

func runKill(args []string, stdout, stderr io.Writer) int {
	return runTransition("kill", (*board.Board).Kill, args, stdout, stderr)
}

func runRevive(args []string, stdout, stderr io.Writer) int {
	return runTransition("revive", (*board.Board).Revive, args, stdout, stderr)
}

// runDone is the only transition handler that doesn't go through
// runTransition: it surfaces the contract checklist before
// committing the move (designs/focus-issue-001.md §"Contract check
// on focus done"). On a TTY we prompt y/N; in non-tty contexts we
// print the contract and require --force.
func runDone(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("done", flag.ContinueOnError)
	fs.SetOutput(stderr)
	force := fs.Bool("force", false, "skip the contract prompt")
	fs.Usage = func() { fmt.Fprintln(stderr, "usage: focus done <id> [--force]") }
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fs.Usage()
		return 2
	}
	id, err := strconv.Atoi(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(stderr, "focus: invalid id %q\n", fs.Arg(0))
		return 2
	}

	b, code := openBoard(stderr)
	if code != 0 {
		return code
	}

	c, _, err := b.LoadCard(id)
	if err != nil {
		fmt.Fprintf(stderr, "focus: %v\n", err)
		return 1
	}

	if len(c.Contract) > 0 && !*force {
		fmt.Fprintln(stdout, "contract:")
		for _, item := range c.Contract {
			fmt.Fprintf(stdout, "  - %s\n", item)
		}
		if !isTTY(os.Stdin) {
			fmt.Fprintln(stderr, "focus: contract has items; rerun with --force in non-interactive contexts")
			return 1
		}
		fmt.Fprint(stdout, "confirm done? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		ans := strings.ToLower(strings.TrimSpace(line))
		if ans != "y" && ans != "yes" {
			fmt.Fprintln(stdout, "aborted")
			return 1
		}
	}

	moved, err := b.Done(id, *force)
	if err != nil {
		fmt.Fprintf(stderr, "focus: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "#%s → %s\n", card.PaddedID(moved.ID), moved.Status)
	return 0
}
