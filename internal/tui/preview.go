package tui

import (
	"fmt"
	"strings"

	"github.com/evanstern/focus/internal/board"
	"github.com/evanstern/focus/internal/board/card"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/glamour"
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
	// wrap width. Width is in the key because glamour reflows on
	// resize and the wrapped output is what we want to cache.
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
		viewport: viewport.New(0, 0),
		rendered: make(map[previewKey]string),
	}
}

// load fetches a card by id. Cheap to call on every cursor move; the
// board package reads from the disk file each time, but we cache the
// glamour render which is the expensive bit.
//
// Forces the viewport to be re-populated on next view(): frontmatter
// fields (status, priority, tags, ...) may have changed even when the
// id is the same (e.g. after a transition like activate/done), and
// the cached viewport content would otherwise stay stale until a
// resize.
//
// Resets the viewport scroll position to top whenever the loaded
// card's id changes — by design, we don't preserve per-card scroll
// across cursor moves. Same-id reloads keep their YOffset so a
// transition doesn't yank the user away from where they were
// reading.
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

// view renders the preview pane to fit width × height. The viewport
// occupies the area inside the (previewPadX, previewPadY) inset; the
// glamour-rendered body wraps to viewport.Width - 2 to match the
// pre-viewport convention.
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
		m.viewport.Width = innerW
		m.viewport.Height = innerH
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
// blank rows at the top and bottom. Used to inset content from a
// surrounding border.
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

// renderBody styles the card body with glamour, wrapping to the
// supplied width. With our paragraph-shaped body convention,
// glamour's word-wrap is the right thing: it reflows paragraphs to
// width while preserving the structural line breaks (lists, code
// fences, headings).
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
	m.rendered[key] = out
	return out
}

// invalidate drops cached renders for this card and resets the
// viewport scroll position to top. Called after transitions or
// edits where the on-disk content may have changed.
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
// viewport's scroll methods so the key handler doesn't have to
// reach through previewModel.viewport directly.
func (m *previewModel) scrollLineDown()     { m.viewport.LineDown(1) }
func (m *previewModel) scrollLineUp()       { m.viewport.LineUp(1) }
func (m *previewModel) scrollHalfPageDown() { m.viewport.HalfPageDown() }
func (m *previewModel) scrollHalfPageUp()   { m.viewport.HalfPageUp() }
func (m *previewModel) scrollPageDown()     { m.viewport.PageDown() }
func (m *previewModel) scrollPageUp()       { m.viewport.PageUp() }
func (m *previewModel) scrollToTop()        { m.viewport.GotoTop() }
func (m *previewModel) scrollToBottom()     { m.viewport.GotoBottom() }
