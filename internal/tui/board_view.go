package tui

import (
	"fmt"
	"strings"

	"github.com/evanstern/focus/internal/board"
	"github.com/evanstern/focus/internal/board/card"
	"github.com/evanstern/focus/internal/board/index"

	tea "github.com/charmbracelet/bubbletea"
)

// boardModel is the default view: a flat list of cards (active
// first, then backlog, then epics) with vim-style cursor movement.
//
// We deliberately don't use bubbles/list — the kanban shape (three
// labeled sections) doesn't map well onto a single list, and we want
// j/k to skip cleanly across sections. A handwritten cursor over a
// flattened slice is simpler than wrangling list.Model into the
// right shape.
type boardModel struct {
	rows     []row
	cursor   int
	wipLimit int
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

// applyReload swaps in the latest BoardView from a reloadedMsg.
// We rebuild the row list rather than mutating in place so the cursor
// can be re-clamped to a valid card index.
func (m *boardModel) applyReload(msg reloadedMsg) {
	m.rows = m.rows[:0]
	m.rows = append(m.rows, row{header: fmt.Sprintf("ACTIVE (%d/%d)", len(msg.view.Active), m.wipLimit)})
	for i := range msg.view.Active {
		e := msg.view.Active[i]
		m.rows = append(m.rows, row{entry: &e})
	}
	m.rows = append(m.rows, row{header: "BACKLOG"})
	for i := range msg.view.Backlog {
		e := msg.view.Backlog[i]
		m.rows = append(m.rows, row{entry: &e})
	}
	if len(msg.view.Epics) > 0 {
		m.rows = append(m.rows, row{header: "EPICS"})
		for i := range msg.view.Epics {
			e := msg.view.Epics[i]
			m.rows = append(m.rows, row{entry: &e})
		}
	}
	m.snapCursorToCard(0)
}

// snapCursorToCard moves m.cursor to the nearest card row >= start.
// Used after reload so the cursor never lands on a header.
func (m *boardModel) snapCursorToCard(start int) {
	if start < 0 {
		start = 0
	}
	for i := start; i < len(m.rows); i++ {
		if m.rows[i].isCard() {
			m.cursor = i
			return
		}
	}
	for i := start - 1; i >= 0; i-- {
		if m.rows[i].isCard() {
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
func (m *boardModel) gotoFirstCard() { m.snapCursorToCard(0) }
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
// in view.
func (m *boardModel) view(width, height int) string {
	if len(m.rows) == 0 {
		return "(loading...)\n"
	}
	if height <= 0 {
		height = 20
	}

	start, end := m.scrollWindow(height)
	var b strings.Builder
	for i := start; i < end; i++ {
		r := m.rows[i]
		if r.header != "" {
			b.WriteString(r.header)
			b.WriteString("\n")
			continue
		}
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}
		b.WriteString(prefix)
		b.WriteString(formatCardRow(*r.entry))
		b.WriteString("\n")
	}
	_ = width
	return b.String()
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

// formatCardRow is the per-card line format. Mirrors the CLI's
// formatRow so users see a consistent layout across surfaces.
func formatCardRow(e index.Entry) string {
	owner := e.Owner
	if owner == "" {
		owner = "-"
	}
	return fmt.Sprintf("#%s  %-40s  %-10s  %-4s  %s",
		card.PaddedID(e.ID), e.Title, e.Project, string(e.Priority), owner,
	)
}

// reloadedMsg is sent by reloadCmd when a fresh BoardView has been
// fetched from disk. The model swaps it in via applyReload.
type reloadedMsg struct {
	view *board.BoardView
}

// statusMsg sets the bottom-bar ephemeral status (e.g. "card #7
// activated"). Cleared by the next reload.
type statusMsg string

// reloadCmd asks Bubble Tea to fetch a fresh BoardView. Wrapped as a
// command so it runs off the message-handling goroutine.
func reloadCmd(b *board.Board) tea.Cmd {
	return func() tea.Msg {
		v, err := b.Board()
		if err != nil {
			return statusMsg("reload error: " + err.Error())
		}
		return reloadedMsg{view: v}
	}
}
