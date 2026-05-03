package tui

import (
	"fmt"
	"strings"

	"github.com/evanstern/focus/internal/board"
	"github.com/evanstern/focus/internal/board/card"
	"github.com/evanstern/focus/internal/board/index"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// boardModel is the default view: a flat list of cards (active
// first, then backlog, then epics) with vim-style cursor movement.
//
// We deliberately don't use bubbles/list — the kanban shape (three
// labeled sections) doesn't map well onto a single list, and we want
// j/k to skip cleanly across sections. A handwritten cursor over a
// flattened slice is simpler than wrangling list.Model into the
// right shape.
// filterMode is which slice of the board the nav pane is showing.
// Tab cycles through these in order; the cycle is closed.
type filterMode int

const (
	filterInFlight filterMode = iota
	filterAll
	filterDone
	filterArchived
)

func (f filterMode) label() string {
	switch f {
	case filterInFlight:
		return "in-flight"
	case filterAll:
		return "all"
	case filterDone:
		return "done"
	case filterArchived:
		return "archived"
	}
	return "?"
}

func (f filterMode) next() filterMode {
	return (f + 1) % 4
}

func (f filterMode) prev() filterMode {
	return (f + 3) % 4
}

type boardModel struct {
	// allRows is the full set of rows from the most recent reload,
	// including section headers and every card in the current filter
	// view. rows is what's actually rendered; when no search query is
	// active rows == allRows. Keeping both lets us build a filtered
	// view without re-fetching from disk on every keystroke.
	allRows  []row
	rows     []row
	cursor   int
	wipLimit int
	filter   filterMode
}

// row is one rendered line in the board: either a header (active /
// backlog / epics label) or a card. cursor only stops on cards.
type row struct {
	header string
	entry  *index.Entry
}

func (r row) isCard() bool { return r.entry != nil }

func newBoardModel(b *board.Board) (boardModel, error) {
	cfg, err := b.LoadConfig()
	if err != nil {
		return boardModel{}, err
	}
	return boardModel{wipLimit: cfg.EffectiveWIPLimit()}, nil
}

// applyReload swaps in the latest data from a reloadedMsg, building
// the all-rows list according to the active filter. The visible
// rows slice is reset to allRows; any in-flight search query needs
// to be reapplied by the caller via applyFilter.
//
// preserveID: if non-zero, attempts to put the cursor on the row
// matching this card id after reload. Falls back to row 0 if the
// id isn't present (filter changed, card archived, etc).
func (m *boardModel) applyReload(msg reloadedMsg, preserveID int) {
	m.filter = msg.filter
	m.allRows = m.allRows[:0]

	if msg.view != nil {
		m.allRows = append(m.allRows, row{header: fmt.Sprintf("ACTIVE (%d/%d)", len(msg.view.Active), m.wipLimit)})
		for i := range msg.view.Active {
			e := msg.view.Active[i]
			m.allRows = append(m.allRows, row{entry: &e})
		}
		m.allRows = append(m.allRows, row{header: "BACKLOG"})
		for i := range msg.view.Backlog {
			e := msg.view.Backlog[i]
			m.allRows = append(m.allRows, row{entry: &e})
		}
		if len(msg.view.Epics) > 0 {
			m.allRows = append(m.allRows, row{header: "EPICS"})
			for i := range msg.view.Epics {
				e := msg.view.Epics[i]
				m.allRows = append(m.allRows, row{entry: &e})
			}
		}
	} else {
		header := strings.ToUpper(msg.filter.label())
		m.allRows = append(m.allRows, row{header: fmt.Sprintf("%s (%d)", header, len(msg.entries))})
		for i := range msg.entries {
			e := msg.entries[i]
			m.allRows = append(m.allRows, row{entry: &e})
		}
	}
	m.rows = m.allRows
	m.placeCursor(preserveID)
}

// applyFilter rebuilds m.rows from m.allRows, keeping only card rows
// whose entry matches the query (case-insensitive substring against
// id, title, project, owner, tags). Section headers are kept only
// if at least one card in their section matches. Empty query leaves
// the full list visible.
//
// If the previously-selected card is still present in the filtered
// view, the cursor stays on it. Otherwise it falls back to the first
// visible card.
func (m *boardModel) applyFilter(query string) {
	prevID := 0
	if e := m.selectedCard(); e != nil {
		prevID = e.ID
	}

	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		m.rows = m.allRows
		m.placeCursor(prevID)
		return
	}

	filtered := make([]row, 0, len(m.allRows))
	pendingHeader := -1
	for i, r := range m.allRows {
		if !r.isCard() {
			pendingHeader = i
			continue
		}
		if !rowMatches(r, q) {
			continue
		}
		if pendingHeader != -1 {
			filtered = append(filtered, m.allRows[pendingHeader])
			pendingHeader = -1
		}
		filtered = append(filtered, r)
	}
	m.rows = filtered
	m.placeCursor(prevID)
}

// rowMatches tests whether a card row matches the search query.
// Searches id (e.g. "0007" or "7"), title, project, owner, tags.
func rowMatches(r row, q string) bool {
	if r.entry == nil {
		return false
	}
	e := r.entry
	hay := strings.ToLower(
		fmt.Sprintf("%04d %d %s %s %s %s",
			e.ID, e.ID, e.Title, e.Project, e.Owner, strings.Join(e.Tags, " "),
		),
	)
	return strings.Contains(hay, q)
}

// placeCursor puts the cursor on the row whose entry id matches
// preserveID. Falls back to the first card if the id isn't present.
func (m *boardModel) placeCursor(preserveID int) {
	if preserveID > 0 {
		for i, r := range m.rows {
			if r.isCard() && r.entry.ID == preserveID {
				m.cursor = i
				return
			}
		}
	}
	m.placeCursorAtFirstCard()
}

func (m *boardModel) placeCursorAtFirstCard() {
	for i, r := range m.rows {
		if r.isCard() {
			m.cursor = i
			return
		}
	}
	m.cursor = 0
}

// moveCursor advances the cursor by delta, skipping header rows.
// Saturates at the first / last card row rather than wrapping.
func (m *boardModel) moveCursor(delta int) {
	step := 1
	if delta < 0 {
		step = -1
		delta = -delta
	}
	for ; delta > 0; delta-- {
		next := m.cursor + step
		// skip headers
		for next >= 0 && next < len(m.rows) && !m.rows[next].isCard() {
			next += step
		}
		if next < 0 || next >= len(m.rows) {
			break
		}
		m.cursor = next
	}
}

// gotoFirstCard / gotoLastCard implement gg / G.
func (m *boardModel) gotoFirstCard() { m.placeCursorAtFirstCard() }
func (m *boardModel) gotoLastCard() {
	for i := len(m.rows) - 1; i >= 0; i-- {
		if m.rows[i].isCard() {
			m.cursor = i
			return
		}
	}
}

// selectedCard returns the entry under the cursor, or nil if the
// list is empty / cursor is on a header (which shouldn't happen
// because moveCursor and snapCursor avoid it).
func (m *boardModel) selectedCard() *index.Entry {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return nil
	}
	return m.rows[m.cursor].entry
}

// view renders the board into a string. height bounds the visible
// rows; if the cursor is offscreen we scroll the window to keep it
// in view. width controls how the per-card row is laid out: the
// title column gets whatever's left after fixed columns, with ellipsis
// truncation if the title overflows.
func (m *boardModel) view(width, height int) string {
	if len(m.rows) == 0 {
		return "(loading...)\n"
	}
	if height <= 0 {
		height = 20
	}
	if width < 20 {
		width = 20
	}

	start, end := m.scrollWindow(height)
	cols := computeColumnWidths(width)

	var b strings.Builder
	for i := start; i < end; i++ {
		r := m.rows[i]

		if i > start && r.header != "" {
			b.WriteString("\n")
		}

		if r.header != "" {
			b.WriteString(styles.header.Render(r.header))
			b.WriteString("\n")
			continue
		}

		line := formatCardRow(*r.entry, cols)
		if i == m.cursor {
			line = styles.cursor.Width(width).Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

// columnWidths describes the width budget allocated to each column
// of a card row. Computed once per view from the available nav width.
//
// Fixed columns (id, project, priority) get fixed budgets. Title gets
// whatever's left. Owner is shown only when the row has at least
// minOwnerWidth columns of slack after the title; otherwise it's
// dropped. This makes the nav responsive: at 80 cols you see all
// columns; at 60 cols owner disappears; at 40 cols title gets very
// short with an ellipsis but the row stays readable.
type columnWidths struct {
	idW       int
	titleW    int
	projectW  int
	priorityW int
	ownerW    int
	gap       int
}

func computeColumnWidths(width int) columnWidths {
	const (
		idW       = 5
		projectW  = 10
		priorityW = 4
		minOwner  = 6
		minTitle  = 10
		gap       = 2
	)
	c := columnWidths{idW: idW, projectW: projectW, priorityW: priorityW, gap: gap}

	fixed := idW + gap + projectW + gap + priorityW

	if width >= fixed+gap+minOwner+gap+minTitle {
		c.ownerW = minOwner
		c.titleW = width - fixed - gap*2 - c.ownerW
		return c
	}

	if width >= fixed+gap+minTitle {
		c.titleW = width - fixed - gap
		return c
	}

	c.titleW = width - idW - gap
	if c.titleW < 1 {
		c.titleW = 1
	}
	c.projectW = 0
	c.priorityW = 0
	return c
}

// formatCardRow renders one card to the column budget in cols. ANSI
// styling for priority is applied last so column-width math (which
// uses ansi.StringWidth) stays correct.
func formatCardRow(e index.Entry, cols columnWidths) string {
	id := "#" + card.PaddedID(e.ID)
	id = padOrTrunc(id, cols.idW)

	title := padOrTrunc(e.Title, cols.titleW)

	var b strings.Builder
	b.WriteString(id)
	b.WriteString(strings.Repeat(" ", cols.gap))
	b.WriteString(title)

	if cols.projectW > 0 {
		b.WriteString(strings.Repeat(" ", cols.gap))
		b.WriteString(padOrTrunc(e.Project, cols.projectW))
	}
	if cols.priorityW > 0 {
		b.WriteString(strings.Repeat(" ", cols.gap))
		prio := padOrTrunc(string(e.Priority), cols.priorityW)
		b.WriteString(priorityStyle(string(e.Priority)).Render(prio))
	}
	if cols.ownerW > 0 {
		owner := e.Owner
		if owner == "" {
			owner = "-"
		}
		b.WriteString(strings.Repeat(" ", cols.gap))
		b.WriteString(padOrTrunc(owner, cols.ownerW))
	}

	return b.String()
}

// padOrTrunc pads s with spaces on the right or truncates with an
// ellipsis to exactly width terminal columns. ANSI-aware.
func padOrTrunc(s string, width int) string {
	if width <= 0 {
		return ""
	}
	w := ansi.StringWidth(s)
	if w == width {
		return s
	}
	if w < width {
		return s + strings.Repeat(" ", width-w)
	}
	if width <= 1 {
		return ansi.Truncate(s, width, "")
	}
	return ansi.Truncate(s, width, "…")
}

// scrollWindow returns the slice [start, end) of m.rows to render
// such that the cursor is visible within height lines.
func (m *boardModel) scrollWindow(height int) (int, int) {
	if len(m.rows) <= height {
		return 0, len(m.rows)
	}
	start := m.cursor - height/2
	if start < 0 {
		start = 0
	}
	end := start + height
	if end > len(m.rows) {
		end = len(m.rows)
		start = end - height
		if start < 0 {
			start = 0
		}
	}
	return start, end
}

// reloadedMsg carries fresh board data and the filter mode it was
// fetched under. Exactly one of view (kanban-shaped) or entries
// (flat list) is non-nil, depending on filter.
//
// preserveID, if non-zero, is the card id the cursor should land on
// after applyReload. Producers set this to whatever card the user
// was looking at (or just edited) so reload doesn't snap the cursor
// back to the top.
type reloadedMsg struct {
	filter     filterMode
	view       *board.BoardView
	entries    []index.Entry
	preserveID int
}

// statusMsg sets the bottom-bar ephemeral status (e.g. "card #7
// activated"). Cleared by the next reload.
type statusMsg string

// reloadCmd fetches the data appropriate for filter f and returns
// a reloadedMsg that places the cursor on preserveID after apply.
// Pass 0 for preserveID to fall back to the first card.
//
// filterInFlight uses Board() (the kanban shape); the others use
// List() with a status filter.
func reloadCmd(b *board.Board, f filterMode, preserveID int) tea.Cmd {
	return func() tea.Msg {
		switch f {
		case filterInFlight:
			v, err := b.Board()
			if err != nil {
				return statusMsg("reload error: " + err.Error())
			}
			return reloadedMsg{filter: f, view: v, preserveID: preserveID}
		default:
			opts := board.ListOpts{}
			switch f {
			case filterDone:
				opts.Status = card.StatusDone
			case filterArchived:
				opts.Status = card.StatusArchived
			}
			es, err := b.List(opts)
			if err != nil {
				return statusMsg("reload error: " + err.Error())
			}
			return reloadedMsg{filter: f, entries: es, preserveID: preserveID}
		}
	}
}
