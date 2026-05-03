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

	// rendered caches the glamour-rendered body keyed by card id.
	// Width is no longer in the key because we don't soft-wrap; the
	// rendered output is the same regardless of how wide the pane is.
	rendered map[previewKey]string
}

type previewKey struct {
	id int
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

// view renders the preview pane to fit width × height. height is a
// hard cap — we truncate output to fit so the layout never breaks.
func (m *previewModel) view(width, height int) string {
	if m.card == nil {
		return ""
	}
	if width < 20 {
		width = 20
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
	b.WriteString(m.renderBody())

	return clipToHeight(b.String(), height)
}

// renderBody styles the card body with glamour but disables soft
// wrap so users see the content exactly as written. Long lines get
// clipped at the pane edge by padPaneLines (using ansi.Truncate).
//
// WithPreservedNewLines is required because glamour's CommonMark
// pass collapses author-inserted line breaks otherwise — paragraphs
// become single long lines, which is the opposite of what we want
// when the on-disk file has hand-formatted line lengths.
//
// The cache is keyed only by card id (not width) since output no
// longer depends on width.
func (m *previewModel) renderBody() string {
	if m.card == nil {
		return ""
	}
	key := previewKey{id: m.card.ID}
	if cached, ok := m.rendered[key]; ok {
		return cached
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(0),
		glamour.WithPreservedNewLines(),
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
