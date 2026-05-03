package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/evanstern/focus/internal/board"
	"github.com/evanstern/focus/internal/board/card"
)

func runNew(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: focus new <title> [--project p] [--priority p0|p1|p2|p3] [--epic id] [--type card|epic] [--slug s]")
	}

	project := fs.String("project", "", "card project")
	priority := fs.String("priority", "p2", "card priority (p0|p1|p2|p3)")
	cardType := fs.String("type", "card", "card type (card|epic)")
	epicID := fs.Int("epic", 0, "set parent epic id (omit for none)")
	slug := fs.String("slug", "", "override the auto-derived slug")

	if err := fs.Parse(reorderFlags(args)); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fs.Usage()
		return 2
	}
	title := strings.Join(fs.Args(), " ")

	b, code := openBoard(stderr)
	if code != 0 {
		return code
	}

	opts := board.NewCardOpts{
		Project:  *project,
		Priority: card.Priority(*priority),
		Type:     card.Type(*cardType),
		Slug:     *slug,
	}
	if *epicID > 0 {
		eid := *epicID
		opts.Epic = &eid
	}

	c, dir, err := b.NewCard(title, opts)
	if err != nil {
		fmt.Fprintf(stderr, "focus: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "%s\n", card.PaddedID(c.ID))
	fmt.Fprintf(stderr, "created %s/%s\n", b.CardsDir(), dir)
	return 0
}
