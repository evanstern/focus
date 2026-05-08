package cli

import (
	"flag"
	"fmt"
	"io"
)

func runBoard(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("board", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprintln(stderr, "usage: focus board [--no-truncate]") }
	noTruncate := fs.Bool("no-truncate", false, "do not truncate titles to terminal width (for piping/scripting)")
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return 2
	}

	b, code := openBoard(stderr)
	if code != 0 {
		return code
	}
	v, err := b.Board()
	if err != nil {
		fmt.Fprintf(stderr, "focus: %v\n", err)
		return 1
	}
	cfg, err := b.LoadConfig()
	if err != nil {
		fmt.Fprintf(stderr, "focus: %v\n", err)
		return 1
	}
	printBoard(stdout, v, cfg.EffectiveWIPLimit(), detectTermWidth(stdout), *noTruncate)
	return 0
}
