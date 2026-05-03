package cli

import (
	"flag"
	"fmt"
	"io"
)

func runReindex(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("reindex", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprintln(stderr, "usage: focus reindex") }
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return 2
	}

	b, code := openBoard(stderr)
	if code != 0 {
		return code
	}
	idx, err := b.Reindex()
	if err != nil {
		fmt.Fprintf(stderr, "focus: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "reindexed: %d cards, next_id=%d\n", len(idx.Cards), idx.NextID)
	return 0
}
