package cli

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/evanstern/focus/internal/board"
)

func runInit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprintln(stderr, "usage: focus init [path]") }
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return 2
	}

	path := "."
	if fs.NArg() > 0 {
		path = fs.Arg(0)
	}

	if path == "." {
		var err error
		path, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "focus: %v\n", err)
			return 1
		}
	}

	b, err := board.Init(path)
	if err != nil {
		fmt.Fprintf(stderr, "focus: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "initialized focus board at %s\n", b.Dir)
	return 0
}
