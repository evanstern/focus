package cli

import (
	"flag"
	"fmt"
	"io"
	"strconv"
)

// runEpic dispatches the `focus epic ...` family. Three subcommands:
//
//   - focus epic <id>            → show progress
//   - focus epic list            → summary of all epics
//   - focus epic add <eid> <cid> → set epic: on a card
func runEpic(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: focus epic <id> | focus epic list | focus epic add <epic-id> <card-id>")
		return 2
	}

	switch args[0] {
	case "list":
		return runEpicList(args[1:], stdout, stderr)
	case "add":
		return runEpicAdd(args[1:], stdout, stderr)
	default:
		return runEpicShow(args, stdout, stderr)
	}
}

func runEpicShow(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("epic", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(stderr, "usage: focus epic <id>")
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
	p, err := b.EpicShow(id)
	if err != nil {
		fmt.Fprintf(stderr, "focus: %v\n", err)
		return 1
	}
	printEpic(stdout, p)
	return 0
}

func runEpicList(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("epic list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return 2
	}
	b, code := openBoard(stderr)
	if code != 0 {
		return code
	}
	eps, err := b.EpicList()
	if err != nil {
		fmt.Fprintf(stderr, "focus: %v\n", err)
		return 1
	}
	printEpicList(stdout, eps)
	return 0
}

func runEpicAdd(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("epic add", flag.ContinueOnError)
	fs.SetOutput(stderr)
	force := fs.Bool("force", false, "skip epic-id existence check")
	fs.Usage = func() { fmt.Fprintln(stderr, "usage: focus epic add <epic-id> <card-id>") }
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return 2
	}
	if fs.NArg() < 2 {
		fs.Usage()
		return 2
	}
	epicID, err := strconv.Atoi(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(stderr, "focus: invalid epic id %q\n", fs.Arg(0))
		return 2
	}
	cardID, err := strconv.Atoi(fs.Arg(1))
	if err != nil {
		fmt.Fprintf(stderr, "focus: invalid card id %q\n", fs.Arg(1))
		return 2
	}

	b, code := openBoard(stderr)
	if code != 0 {
		return code
	}
	if err := b.EpicAdd(epicID, cardID, *force); err != nil {
		fmt.Fprintf(stderr, "focus: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "card #%04d → epic #%04d\n", cardID, epicID)
	return 0
}
