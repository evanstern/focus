package tui

import (
	"fmt"
	"strings"

	"github.com/evanstern/focus/internal/board"
	"github.com/evanstern/focus/internal/board/card"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/glamour"
)

// detailModel renders one card: header (typed frontmatter fields)
// plus the markdown body, scrollable via a bubbles/viewport. We
// use glamour to style the body and cache the rendered output by
// card id (designs/focus-issue-001.md §"Glamour width gotcha").
type detailModel struct {
	board    *board.Board
	card     *card.Card
	dir      string
	viewport viewport.Model

	// rendered caches the glamour-rendered body keyed by card id.
	// Re-rendering on every keystroke is noticeable on cards with code
	// blocks or tables (per the design issue).
	rendered map[int]string

	// width and height are the most recent terminal dimensions;
	// stashed so resize() can rebuild the viewport.
	width, height int
}

func newDetailModel(b *board.Board) detailModel {
	vp := viewport.New(80, 20)
	return detailModel{
		board:    b,
		viewport: vp,
		rendered: make(map[int]string),
	}
}

// resize updates the viewport to match the terminal size. Called
// from the root model on tea.WindowSizeMsg.
func (m *detailModel) resize(w, h int) {
	m.width = w
	m.height = h
	headerHeight := 8
	m.viewport.Width = w
	m.viewport.Height = max(1, h-headerHeight-statusBarHeight)
	if m.card != nil {
		m.viewport.SetContent(m.renderBody())
	}
}

// load fetches a card by id and prepares the viewport. Called on
// "enter" in the board view.
func (m *detailModel) load(id int) error {
	c, dir, err := m.board.LoadCard(id)
	if err != nil {
		return err
	}
	m.card = c
	m.dir = dir
	m.viewport.SetContent(m.renderBody())
	m.viewport.GotoTop()
	return nil
}

// renderBody returns the glamour-rendered body for the current card.
// The width gotcha: glamour must be told the available width minus
// border + padding, otherwise lines overflow the viewport. See
// designs/focus-issue-001.md §"Glamour width gotcha".
func (m *detailModel) renderBody() string {
	if m.card == nil {
		return ""
	}
	if cached, ok := m.rendered[m.card.ID]; ok {
		return cached
	}
	w := m.viewport.Width - 4
	if w < 20 {
		w = 20
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(w),
	)
	if err != nil {
		return m.card.Body
	}
	out, err := r.Render(m.card.Body)
	if err != nil {
		return m.card.Body
	}
	m.rendered[m.card.ID] = out
	return out
}

func (m *detailModel) view() string {
	if m.card == nil {
		return "(no card loaded)\n"
	}
	c := m.card
	var b strings.Builder
	fmt.Fprintf(&b, "#%s  %s\n", card.PaddedID(c.ID), c.Title)
	fmt.Fprintf(&b, "  status: %s   priority: %s   project: %s   type: %s\n",
		c.Status, c.Priority, c.Project, c.Type)
	if c.Epic != nil {
		fmt.Fprintf(&b, "  epic: #%04d\n", *c.Epic)
	}
	if !c.Created.IsZero() {
		fmt.Fprintf(&b, "  created: %s\n", c.Created.Format("2006-01-02"))
	}
	fmt.Fprintf(&b, "  uuid: %s\n", c.UUID)
	if len(c.Contract) > 0 {
		fmt.Fprintln(&b, "  contract:")
		for _, item := range c.Contract {
			fmt.Fprintf(&b, "    - %s\n", item)
		}
	}
	b.WriteString("\n")
	b.WriteString(m.viewport.View())
	return b.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
