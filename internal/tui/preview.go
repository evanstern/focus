package tui

import (
	"fmt"
	"strings"

	"github.com/evanstern/focus/internal/board"
	"github.com/evanstern/focus/internal/board/card"

	"github.com/charmbracelet/glamour"
)

// previewModel renders a single card's frontmatter header + body
// alongside the nav pane. Auto-fits to the available width and
// height; no scrolling. Long bodies clip — that's what `e` (open in
// $EDITOR) is for.
type previewModel struct {
	board *board.Board
	card  *card.Card

	// rendered caches the glamour-rendered body keyed by card id +
	// wrap width. Width is in the key because glamour reflows on
	// resize and the wrapped output is what we want to cache.
	rendered map[previewKey]string
}

type previewKey struct {
	id    int
	width int
}

func newPreviewModel(b *board.Board) previewModel {
	return previewModel{board: b, rendered: make(map[previewKey]string)}
}

// load fetches a card by id. Cheap to call on every cursor move; the
// board package reads from the disk file each time, but we cache the
// glamour render which is the expensive bit.
func (m *previewModel) load(id int) error {
	c, _, err := m.board.LoadCard(id)
	if err != nil {
		return err
	}
	m.card = c
	return nil
}

// previewPadX and previewPadY are the interior padding inside the
// preview pane (between border and content). Cells, not pixels.
const (
	previewPadX = 2
	previewPadY = 1
)

// view renders the preview pane to fit width × height. The body is
// wrapped to (width - 2*previewPadX) so paragraph-shaped on-disk
// content reflows for the available pane size, with breathing room
// inside the border.
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

	return padInsidePane(clipToHeight(b.String(), innerH), previewPadX, previewPadY)
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
		blank := strings.Repeat("", 0)
		var top, bottom []string
		for i := 0; i < padY; i++ {
			top = append(top, blank)
			bottom = append(bottom, blank)
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

// invalidate drops cached renders for this card. Called after
// transitions or edits where the on-disk content may have changed.
func (m *previewModel) invalidate(id int) {
	for k := range m.rendered {
		if k.id == id {
			delete(m.rendered, k)
		}
	}
}

// clipToHeight returns at most n lines of s. Used to keep the
// preview pane from overflowing the layout when the body is long.
func clipToHeight(s string, n int) string {
	if n <= 0 {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[:n], "\n")
}
