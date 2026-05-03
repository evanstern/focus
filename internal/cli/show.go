package cli

import (
	"flag"
	"fmt"
	"io"
	"strconv"
)

func runShow(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprintln(stderr, "usage: focus show <id>") }
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
	c, dir, err := b.LoadCard(id)
	if err != nil {
		fmt.Fprintf(stderr, "focus: %v\n", err)
		return 1
	}
	printShow(stdout, c, dir)
	return 0
}
