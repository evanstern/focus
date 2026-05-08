package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/x/term"
	"github.com/evanstern/focus/internal/board"
	"github.com/evanstern/focus/internal/board/card"
	"github.com/evanstern/focus/internal/board/index"
)

// Row layout constants for the columnar list/board format. Used to
// compute how many columns are left for the title.
//
//	#XXXX  TITLE  PROJECT(10)  PRIORITY(4)  OWNER
//	 4  2     2    10        2     4       2  …
//
// The id column is always 4 runes (PaddedID), project is %-10s, and
// priority is %-4s. Owner is variable but appended last so it can
// drift without disturbing the columns to its left.
const (
	rowIDWidth       = 4  // PaddedID width
	rowProjectWidth  = 10 // %-10s
	rowPriorityWidth = 4  // %-4s
	rowSeparator     = 2  // "  " between every column

	// Fixed cost in runes for everything in formatRow except the
	// title and the trailing owner: `#` + id + sep + sep + project
	// + sep + priority + sep = 1 + 4 + 2 + 2 + 10 + 2 + 4 + 2 = 27.
	rowFixedCols = 1 + rowIDWidth + rowSeparator + rowSeparator +
		rowProjectWidth + rowSeparator + rowPriorityWidth + rowSeparator

	// Minimum runes we'll allocate to a title even on absurdly narrow
	// terminals. Below this we accept the wrap.
	minTitleBudget = 20

	// Default fallback width when stdout/stderr aren't a tty or the
	// detection call errors. 80 is the universal terminal contract.
	defaultTermWidth = 80
)

// detectTermWidth returns the width of the controlling terminal in
// columns. It tries stdout first, then stderr (whichever is a tty).
// Falls back to defaultTermWidth on any error, non-tty, or pipe.
//
// Encapsulating the call here lets tests bypass it by calling
// formatRowWidth directly with a fixed width.
func detectTermWidth() int {
	for _, f := range []*os.File{os.Stdout, os.Stderr} {
		if f == nil {
			continue
		}
		fd := f.Fd()
		if !term.IsTerminal(fd) {
			continue
		}
		w, _, err := term.GetSize(fd)
		if err != nil || w <= 0 {
			continue
		}
		return w
	}
	return defaultTermWidth
}

// printBoard renders the default board view (active + backlog +
// epics) in the columnar format the design doc sketches. No color in
// v0.1.0; styling lives in the TUI.
func printBoard(w io.Writer, v *board.BoardView, wipLimit int, termWidth int, noTruncate bool) {
	// Board rows are indented two spaces; subtract that from the
	// width budget so the title column lines up with the rest.
	rowWidth := termWidth - 2

	fmt.Fprintf(w, "ACTIVE (%d/%d)\n", len(v.Active), wipLimit)
	if len(v.Active) == 0 {
		fmt.Fprintln(w, "  (none)")
	} else {
		for _, e := range v.Active {
			fmt.Fprintln(w, "  "+formatRowWidth(e, rowWidth, noTruncate))
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "BACKLOG")
	if len(v.Backlog) == 0 {
		fmt.Fprintln(w, "  (none)")
	} else {
		for _, e := range v.Backlog {
			fmt.Fprintln(w, "  "+formatRowWidth(e, rowWidth, noTruncate))
		}
	}

	if len(v.Epics) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "EPICS")
		for _, e := range v.Epics {
			fmt.Fprintln(w, "  "+formatRowWidth(e, rowWidth, noTruncate))
		}
	}
}

// formatRow renders a single index Entry as one terminal line using
// the auto-detected terminal width. Thin wrapper preserved for
// callers that don't thread a width through.
func formatRow(e index.Entry) string {
	return formatRowWidth(e, detectTermWidth(), false)
}

// formatRowWidth renders one row at a target terminal width. If
// noTruncate is true (or width <= 0), the title is not truncated and
// columns may drift on long titles — caller's choice, e.g. for
// piping. Otherwise the title is truncated to fit the remaining
// budget after fixed columns, with a single-rune `…` (U+2026)
// marker. Truncation works on runes, not bytes, so non-ASCII titles
// (emoji, accents) cut at character boundaries.
//
// Silent truncation is hostile, but `…` is loud: the user can see
// the row was clipped and ask for `--no-truncate` if they want the
// full title.
func formatRowWidth(e index.Entry, width int, noTruncate bool) string {
	owner := e.Owner
	if owner == "" {
		owner = "-"
	}

	if noTruncate || width <= 0 {
		// Legacy behavior: %-40s pads short titles, never truncates
		// long ones. Columns drift on long titles. Documented for
		// scripting / piping use.
		return fmt.Sprintf("#%s  %-40s  %-10s  %-4s  %s",
			card.PaddedID(e.ID), e.Title, e.Project, string(e.Priority), owner,
		)
	}

	// Compute the title budget. The owner column is variable and
	// trailing, so it's part of the cost on the right side.
	budget := width - rowFixedCols - utf8.RuneCountInString(owner)
	if budget < minTitleBudget {
		budget = minTitleBudget
	}

	title := truncateRunes(e.Title, budget)
	// Pad the title to the budget so columns line up across rows.
	padded := title + strings.Repeat(" ", budget-utf8.RuneCountInString(title))

	return fmt.Sprintf("#%s  %s  %-10s  %-4s  %s",
		card.PaddedID(e.ID), padded, e.Project, string(e.Priority), owner,
	)
}

// truncateRunes returns s truncated to at most max runes, replacing
// the trailing rune with `…` (U+2026) if truncation occurred. If s
// already fits, it's returned unchanged. max < 1 returns "".
func truncateRunes(s string, max int) string {
	if max < 1 {
		return ""
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	// Take the first max-1 runes, then append the ellipsis. We walk
	// the string by rune (range over string) rather than slicing by
	// byte so multi-byte characters cut cleanly.
	var b strings.Builder
	count := 0
	for _, r := range s {
		if count >= max-1 {
			break
		}
		b.WriteRune(r)
		count++
	}
	b.WriteRune('…')
	return b.String()
}

// printList renders a flat list of entries (one per line) for `focus
// list`. Same row format as the board, with a leading status column.
func printList(w io.Writer, entries []index.Entry, termWidth int, noTruncate bool) {
	if len(entries) == 0 {
		fmt.Fprintln(w, "(no cards match)")
		return
	}
	// "STATUS    " is 9 + 2 = 11 runes before the row begins.
	rowWidth := termWidth - 11
	for _, e := range entries {
		fmt.Fprintf(w, "%-9s  %s\n", string(e.Status), formatRowWidth(e, rowWidth, noTruncate))
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
// list`. Layout: `#XXXX  TITLE  N/M done`. We truncate the title the
// same way as formatRowWidth.
func printEpicList(w io.Writer, eps []board.EpicProgress, termWidth int, noTruncate bool) {
	if len(eps) == 0 {
		fmt.Fprintln(w, "(no epics)")
		return
	}
	for _, p := range eps {
		total := p.Total()
		progress := fmt.Sprintf("%d/%d done", p.Done, total)

		if noTruncate || termWidth <= 0 {
			fmt.Fprintf(w, "#%s  %-40s  %s\n",
				card.PaddedID(p.Epic.ID), p.Epic.Title, progress)
			continue
		}

		// Fixed cost: `#` + id(4) + sep + sep + progress + sep
		// (trailing newline handled by Fprintln).
		fixed := 1 + rowIDWidth + rowSeparator + rowSeparator +
			utf8.RuneCountInString(progress)
		budget := termWidth - fixed
		if budget < minTitleBudget {
			budget = minTitleBudget
		}
		title := truncateRunes(p.Epic.Title, budget)
		padded := title + strings.Repeat(" ", budget-utf8.RuneCountInString(title))

		fmt.Fprintf(w, "#%s  %s  %s\n",
			card.PaddedID(p.Epic.ID), padded, progress)
	}
}
