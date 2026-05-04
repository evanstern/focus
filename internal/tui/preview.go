package tui

import (
	"fmt"
	"strings"

	"github.com/evanstern/focus/internal/board"
	"github.com/evanstern/focus/internal/board/card"

	"charm.land/bubbles/v2/viewport"
	"charm.land/glamour/v2"
)

// previewModel renders a single card's frontmatter header + body in
// a scrollable viewport alongside the nav pane. When the preview
// pane has keyboard focus, vim scroll keys move the viewport's
// YOffset; otherwise it just sits at whatever offset it was last
// scrolled to (and is reset to top whenever a different card is
// loaded).
type previewModel struct {
	board *board.Board
	card  *card.Card

	// viewport owns the scroll bounds + half/full-page math. We
	// SetContent on every load and on every resize (because glamour
	// reflows on width changes and so does the header width math).
	viewport viewport.Model

	// rendered caches the glamour-rendered body keyed by card id +
	// wrap width.
	rendered map[previewKey]string

	// lastWidth/lastHeight track what we last sized the viewport to,
	// so we only re-set content when the layout actually changes.
	lastWidth  int
	lastHeight int
}

type previewKey struct {
	id    int
	width int
}

func newPreviewModel(b *board.Board) previewModel {
	return previewModel{
		board:    b,
		viewport: viewport.New(),
		rendered: make(map[previewKey]string),
	}
}

// load fetches a card by id. Resets the viewport scroll position to
// top whenever the loaded card's id changes; same-id reloads keep
// their YOffset so a transition doesn't yank the user away from
// where they were reading.
func (m *previewModel) load(id int) error {
	c, _, err := m.board.LoadCard(id)
	if err != nil {
		return err
	}
	prevID := 0
	if m.card != nil {
		prevID = m.card.ID
	}
	m.card = c
	m.lastWidth = 0
	m.lastHeight = 0
	if prevID != c.ID {
		m.viewport.SetYOffset(0)
	}
	return nil
}

// previewPadX and previewPadY are the interior padding inside the
// preview pane (between border and content). Cells, not pixels.
const (
	previewPadX = 2
	previewPadY = 1
)

// view renders the preview pane to fit width × height.
func (m *previewModel) view(width, height int) string {
	if m.card == nil {
		return ""
	}
	if width < 20 {
		width = 20
	}

	innerW := width - 2*previewPadX
	if innerW < 10 {
		innerW = 10
	}
	innerH := height - 2*previewPadY
	if innerH < 1 {
		innerH = 1
	}

	if innerW != m.lastWidth || innerH != m.lastHeight {
		m.viewport.SetWidth(innerW)
		m.viewport.SetHeight(innerH)
		m.viewport.SetContent(m.buildContent(innerW))
		m.lastWidth = innerW
		m.lastHeight = innerH
	}

	return padInsidePane(m.viewport.View(), previewPadX, previewPadY)
}

// buildContent assembles the full scrollable string for the viewport:
// frontmatter header followed by the glamour-rendered body. innerW
// is the viewport's content width.
func (m *previewModel) buildContent(innerW int) string {
	var b strings.Builder
	c := m.card
	fmt.Fprintf(&b, "#%s  %s\n", card.PaddedID(c.ID), c.Title)
	fmt.Fprintf(&b, "  status: %s   priority: %s\n", c.Status, c.Priority)
	fmt.Fprintf(&b, "  project: %s   type: %s\n", c.Project, c.Type)
	if c.Epic != nil {
		fmt.Fprintf(&b, "  epic: #%04d\n", *c.Epic)
	}
	if !c.Created.IsZero() {
		fmt.Fprintf(&b, "  created: %s\n", c.Created.Format("2006-01-02"))
	}
	if c.Owner != "" {
		fmt.Fprintf(&b, "  owner: %s\n", c.Owner)
	}
	if len(c.Tags) > 0 {
		fmt.Fprintf(&b, "  tags: %s\n", strings.Join(c.Tags, ", "))
	}
	if len(c.Contract) > 0 {
		fmt.Fprintln(&b, "  contract:")
		for _, item := range c.Contract {
			fmt.Fprintf(&b, "    - %s\n", item)
		}
	}
	b.WriteString("\n")
	b.WriteString(m.renderBody(innerW))
	return b.String()
}

// padInsidePane prepends padX spaces to every line and adds padY
// blank rows at the top and bottom.
func padInsidePane(s string, padX, padY int) string {
	if padX == 0 && padY == 0 {
		return s
	}
	xPad := strings.Repeat(" ", padX)
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = xPad + line
	}
	if padY > 0 {
		var top, bottom []string
		for i := 0; i < padY; i++ {
			top = append(top, "")
			bottom = append(bottom, "")
		}
		lines = append(top, append(lines, bottom...)...)
	}
	return strings.Join(lines, "\n")
}

// renderBody styles the card body with glamour, wrapping to width.
// Glamour v2 dropped WithAutoStyle; "dark" is the default and matches
// the lipgloss palette we use elsewhere.
func (m *previewModel) renderBody(width int) string {
	if m.card == nil {
		return ""
	}
	w := width - 2
	if w < 20 {
		w = 20
	}
	key := previewKey{id: m.card.ID, width: w}
	if cached, ok := m.rendered[key]; ok {
		return cached
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(w),
	)
	if err != nil {
		return m.card.Body
	}
	out, err := r.Render(m.card.Body)
	if err != nil {
		return m.card.Body
	}
	m.rendered[key] = out
	return out
}

// invalidate drops cached renders for this card and resets the
// viewport scroll position to top.
func (m *previewModel) invalidate(id int) {
	for k := range m.rendered {
		if k.id == id {
			delete(m.rendered, k)
		}
	}
	m.lastWidth = 0
	m.lastHeight = 0
	m.viewport.SetYOffset(0)
}

// scrollLineDown / scrollLineUp / scrollHalfPageDown etc. wrap the
// viewport's scroll methods.
func (m *previewModel) scrollLineDown()     { m.viewport.ScrollDown(1) }
func (m *previewModel) scrollLineUp()       { m.viewport.ScrollUp(1) }
func (m *previewModel) scrollHalfPageDown() { m.viewport.HalfPageDown() }
func (m *previewModel) scrollHalfPageUp()   { m.viewport.HalfPageUp() }
func (m *previewModel) scrollPageDown()     { m.viewport.PageDown() }
func (m *previewModel) scrollPageUp()       { m.viewport.PageUp() }
func (m *previewModel) scrollToTop()        { m.viewport.GotoTop() }
func (m *previewModel) scrollToBottom()     { m.viewport.GotoBottom() }
