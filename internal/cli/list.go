package cli

import (
	"flag"
	"fmt"
	"io"

	"github.com/evanstern/focus/internal/board"
	"github.com/evanstern/focus/internal/board/card"
)

func runList(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: focus list [status] [--project p] [--priority p0|p1|p2|p3] [--epic id] [--owner o] [--tag t] [--type card|epic]")
	}

	project := fs.String("project", "", "filter by project")
	priority := fs.String("priority", "", "filter by priority")
	owner := fs.String("owner", "", "filter by owner")
	tag := fs.String("tag", "", "filter by tag")
	cardType := fs.String("type", "", "filter by type (card|epic)")
	epicID := fs.Int("epic", 0, "filter by epic id (0 = no filter)")

	if err := fs.Parse(reorderFlags(args)); err != nil {
		return 2
	}

	opts := board.ListOpts{
		Project:  *project,
		Priority: card.Priority(*priority),
		Owner:    *owner,
		Tag:      *tag,
		Type:     card.Type(*cardType),
	}
	if *epicID > 0 {
		eid := *epicID
		opts.Epic = &eid
	}
	if fs.NArg() > 0 {
		opts.Status = card.Status(fs.Arg(0))
	}

	b, code := openBoard(stderr)
	if code != 0 {
		return code
	}
	entries, err := b.List(opts)
	if err != nil {
		fmt.Fprintf(stderr, "focus: %v\n", err)
		return 1
	}
	printList(stdout, entries)
	return 0
}
