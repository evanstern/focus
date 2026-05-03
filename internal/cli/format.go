package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/evanstern/focus/internal/board"
	"github.com/evanstern/focus/internal/board/card"
	"github.com/evanstern/focus/internal/board/index"
)

// printBoard renders the default board view (active + backlog +
// epics) in the columnar format the design doc sketches. No color in
// v0.1.0; styling lives in the TUI.
func printBoard(w io.Writer, v *board.BoardView, wipLimit int) {
	fmt.Fprintf(w, "ACTIVE (%d/%d)\n", len(v.Active), wipLimit)
	if len(v.Active) == 0 {
		fmt.Fprintln(w, "  (none)")
	} else {
		for _, e := range v.Active {
			fmt.Fprintln(w, "  "+formatRow(e))
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "BACKLOG")
	if len(v.Backlog) == 0 {
		fmt.Fprintln(w, "  (none)")
	} else {
		for _, e := range v.Backlog {
			fmt.Fprintln(w, "  "+formatRow(e))
		}
	}

	if len(v.Epics) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "EPICS")
		for _, e := range v.Epics {
			fmt.Fprintln(w, "  "+formatRow(e))
		}
	}
}

// formatRow renders a single index Entry as one terminal line. Format:
//
//	#0142  Ship the feature                    api      p2     ash
//
// Columns are space-padded for casual reading. We don't wrap or
// truncate — terminals reflow on their own and truncating titles
// silently is hostile.
func formatRow(e index.Entry) string {
	owner := e.Owner
	if owner == "" {
		owner = "-"
	}
	return fmt.Sprintf("#%s  %-40s  %-10s  %-4s  %s",
		card.PaddedID(e.ID), e.Title, e.Project, string(e.Priority), owner,
	)
}

// printList renders a flat list of entries (one per line) for `focus
// list`. Same row format as the board.
func printList(w io.Writer, entries []index.Entry) {
	if len(entries) == 0 {
		fmt.Fprintln(w, "(no cards match)")
		return
	}
	for _, e := range entries {
		fmt.Fprintf(w, "%-9s  %s\n", string(e.Status), formatRow(e))
	}
}

// printShow renders a card's detail view: a header with the typed
// frontmatter fields followed by the body verbatim. We do NOT
// markdown-render in the CLI — the TUI uses glamour, but the CLI
// keeps things grep-friendly.
func printShow(w io.Writer, c *card.Card, dirName string) {
	fmt.Fprintf(w, "#%s  %s\n", card.PaddedID(c.ID), c.Title)
	fmt.Fprintf(w, "  status:    %s\n", c.Status)
	fmt.Fprintf(w, "  priority:  %s\n", c.Priority)
	fmt.Fprintf(w, "  type:      %s\n", c.Type)
	fmt.Fprintf(w, "  project:   %s\n", c.Project)
	if c.Epic != nil {
		fmt.Fprintf(w, "  epic:      %d\n", *c.Epic)
	}
	if c.Owner != "" {
		fmt.Fprintf(w, "  owner:     %s\n", c.Owner)
	}
	if len(c.Tags) > 0 {
		fmt.Fprintf(w, "  tags:      %s\n", strings.Join(c.Tags, ", "))
	}
	if len(c.Contract) > 0 {
		fmt.Fprintln(w, "  contract:")
		for _, item := range c.Contract {
			fmt.Fprintf(w, "    - %s\n", item)
		}
	}
	if !c.Created.IsZero() {
		fmt.Fprintf(w, "  created:   %s\n", c.Created.Format("2006-01-02"))
	}
	fmt.Fprintf(w, "  uuid:      %s\n", c.UUID)
	fmt.Fprintf(w, "  dir:       %s\n", dirName)
	fmt.Fprintln(w)
	fmt.Fprintln(w, strings.TrimRight(c.Body, "\n"))
}

// printEpic renders the progress summary for one epic.
func printEpic(w io.Writer, p *board.EpicProgress) {
	e := p.Epic
	fmt.Fprintf(w, "#%s  %s  [epic]\n", card.PaddedID(e.ID), e.Title)
	fmt.Fprintf(w, "  status:   %s\n", e.Status)
	fmt.Fprintf(w, "  project:  %s\n", e.Project)
	if e.Description != "" {
		fmt.Fprintf(w, "  desc:     %s\n", e.Description)
	}
	fmt.Fprintln(w)
	total := p.Total()
	if total == 0 {
		fmt.Fprintln(w, "  (no child cards yet)")
		return
	}
	fmt.Fprintf(w, "  progress: %d/%d done\n", p.Done, total)
	fmt.Fprintf(w, "    active:   %d\n", p.Active)
	fmt.Fprintf(w, "    backlog:  %d\n", p.Backlog)
	fmt.Fprintf(w, "    done:     %d\n", p.Done)
	if p.Archive > 0 {
		fmt.Fprintf(w, "    archived: %d\n", p.Archive)
	}
}

// printEpicList renders a one-line-per-epic summary for `focus epic
// list`.
func printEpicList(w io.Writer, eps []board.EpicProgress) {
	if len(eps) == 0 {
		fmt.Fprintln(w, "(no epics)")
		return
	}
	for _, p := range eps {
		total := p.Total()
		fmt.Fprintf(w, "#%s  %-40s  %d/%d done\n",
			card.PaddedID(p.Epic.ID), p.Epic.Title, p.Done, total)
	}
}
