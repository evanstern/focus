package tui

import (
	"strings"
	"testing"

	"github.com/evanstern/focus/internal/board"

	tea "charm.land/bubbletea/v2"
)

func setupBoard(t *testing.T) *board.Board {
	t.Helper()
	root := t.TempDir()
	b, err := board.Init(root)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	for _, title := range []string{"alpha", "beta", "gamma"} {
		if _, _, err := b.NewCard(title, board.NewCardOpts{}); err != nil {
			t.Fatalf("NewCard %s: %v", title, err)
		}
	}
	return b
}

// runeKey builds a printable-rune key press message.
func runeKey(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Text: string(r)}
}

func typeString(t *testing.T, m *Model, s string) *Model {
	t.Helper()
	for _, r := range s {
		updated, _ := m.Update(runeKey(r))
		m = updated.(*Model)
	}
	return m
}

func TestModelBoardCursorMovement(t *testing.T) {
	b := setupBoard(t)
	m, err := newModel(b)
	if err != nil {
		t.Fatal(err)
	}
	v, err := b.Board()
	if err != nil {
		t.Fatal(err)
	}
	updated, _ := m.Update(reloadedMsg{view: v})
	m = updated.(*Model)

	first := m.board_.selectedCard()
	if first == nil {
		t.Fatal("no card under cursor after reload")
	}

	updated, _ = m.Update(runeKey('j'))
	m = updated.(*Model)
	second := m.board_.selectedCard()
	if second == nil || second.ID == first.ID {
		t.Errorf("j didn't move cursor; first=%v second=%v", first, second)
	}

	updated, _ = m.Update(runeKey('k'))
	m = updated.(*Model)
	if got := m.board_.selectedCard(); got == nil || got.ID != first.ID {
		t.Errorf("k didn't move cursor back; got=%v", got)
	}
}

func TestPreviewLoadsOnCursorMove(t *testing.T) {
	b := setupBoard(t)
	m, _ := newModel(b)
	v, _ := b.Board()
	updated, _ := m.Update(reloadedMsg{view: v})
	m = updated.(*Model)

	if m.preview.card == nil {
		t.Fatal("preview not loaded after initial reload")
	}
	first := m.preview.card.ID

	updated, _ = m.Update(runeKey('j'))
	m = updated.(*Model)
	if m.preview.card == nil || m.preview.card.ID == first {
		t.Errorf("preview didn't follow cursor on j; got=%v", m.preview.card)
	}
}

func TestEnterInvokesEditor(t *testing.T) {
	b := setupBoard(t)
	m, _ := newModel(b)
	v, _ := b.Board()
	updated, _ := m.Update(reloadedMsg{view: v})
	m = updated.(*Model)

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter produced no command")
	}
}

func TestModelHelpToggle(t *testing.T) {
	b := setupBoard(t)
	m, _ := newModel(b)
	updated, _ := m.Update(runeKey('?'))
	m = updated.(*Model)
	if m.view != viewHelp {
		t.Errorf("view after ? = %v, want help", m.view)
	}
	helpStr := m.help.View(m.keys)
	if helpStr == "" {
		t.Error("help view is empty")
	}
	updated, _ = m.Update(runeKey('?'))
	m = updated.(*Model)
	if m.view != viewBoard {
		t.Errorf("? again should toggle back to board, got %v", m.view)
	}
}

func TestModelSearchFiltersAndJumps(t *testing.T) {
	b := setupBoard(t)
	m, _ := newModel(b)
	v, _ := b.Board()
	updated, _ := m.Update(reloadedMsg{view: v})
	m = updated.(*Model)

	updated, _ = m.Update(runeKey('/'))
	m = updated.(*Model)
	if m.input != modeSearch {
		t.Fatalf("input = %v, want search", m.input)
	}
	m = typeString(t, m, "gamma")
	if got := m.board_.selectedCard(); got == nil || got.Title != "gamma" {
		t.Errorf("live search didn't jump to gamma; got %+v", got)
	}

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(*Model)
	if m.input != modeNormal {
		t.Errorf("enter should exit search mode, got %v", m.input)
	}
	if got := m.board_.selectedCard(); got == nil || got.Title != "gamma" {
		t.Errorf("filter cleared on enter; got %+v", got)
	}
}

func TestSearchByCardID(t *testing.T) {
	b := setupBoard(t)
	m, _ := newModel(b)
	v, _ := b.Board()
	updated, _ := m.Update(reloadedMsg{view: v})
	m = updated.(*Model)

	updated, _ = m.Update(runeKey('/'))
	m = updated.(*Model)
	m = typeString(t, m, "0002")
	got := m.board_.selectedCard()
	if got == nil || got.ID != 2 {
		t.Errorf("search /0002 should find card #2; got %+v", got)
	}
}

func TestSearchEscClearsFilter(t *testing.T) {
	b := setupBoard(t)
	m, _ := newModel(b)
	v, _ := b.Board()
	updated, _ := m.Update(reloadedMsg{view: v})
	m = updated.(*Model)

	updated, _ = m.Update(runeKey('/'))
	m = updated.(*Model)
	m = typeString(t, m, "gamma")
	cardCount := func() int {
		n := 0
		for _, r := range m.board_.rows {
			if r.isCard() {
				n++
			}
		}
		return n
	}
	if cardCount() != 1 {
		t.Fatalf("expected 1 card visible after filter; got %d", cardCount())
	}

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = updated.(*Model)
	if cardCount() != 3 {
		t.Errorf("esc should restore full list; got %d cards", cardCount())
	}
	if m.status != "" {
		t.Errorf("esc didn't clear status: %q", m.status)
	}
}

func TestSearchNoMatchesStatusClearsOnBackspace(t *testing.T) {
	b := setupBoard(t)
	m, _ := newModel(b)
	v, _ := b.Board()
	updated, _ := m.Update(reloadedMsg{view: v})
	m = updated.(*Model)

	updated, _ = m.Update(runeKey('/'))
	m = updated.(*Model)
	m = typeString(t, m, "zzzzz")
	if m.status != "no matches" {
		t.Errorf("expected 'no matches' status; got %q", m.status)
	}
	for i := 0; i < 5; i++ {
		updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
		m = updated.(*Model)
	}
	if m.status != "" {
		t.Errorf("status should clear when filter clears; got %q", m.status)
	}
}

func TestEditPreservesCursor(t *testing.T) {
	b := setupBoard(t)
	m, _ := newModel(b)
	v, _ := b.Board()
	updated, _ := m.Update(reloadedMsg{view: v})
	m = updated.(*Model)

	for i := 0; i < 2; i++ {
		updated, _ = m.Update(runeKey('j'))
		m = updated.(*Model)
	}
	beforeID := m.board_.selectedCard().ID

	updated, _ = m.Update(editFinishedMsg{id: beforeID})
	m = updated.(*Model)

	v2, _ := b.Board()
	updated, _ = m.Update(reloadedMsg{view: v2, preserveID: beforeID})
	m = updated.(*Model)

	got := m.board_.selectedCard()
	if got == nil || got.ID != beforeID {
		t.Errorf("cursor lost after reload-with-preserve; want id=%d, got=%+v", beforeID, got)
	}
}

func TestCommandModeQuitsViaQ(t *testing.T) {
	b := setupBoard(t)
	m, _ := newModel(b)
	updated, _ := m.Update(runeKey(':'))
	m = updated.(*Model)
	if m.input != modeCommand {
		t.Fatalf("input = %v, want command", m.input)
	}
	updated, cmd := m.Update(runeKey('q'))
	m = updated.(*Model)
	updated, cmd2 := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(*Model)
	if cmd == nil && cmd2 == nil {
		t.Error("command :q didn't produce a Quit cmd")
	}
}

func TestCommandModeNewCreatesCard(t *testing.T) {
	b := setupBoard(t)
	m, _ := newModel(b)

	updated, _ := m.Update(runeKey(':'))
	m = updated.(*Model)
	m = typeString(t, m, "new fresh card")
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal(":new produced no command")
	}
	msg := cmd()
	if _, ok := msg.(reloadedMsg); !ok {
		t.Errorf(":new returned %T, want reloadedMsg", msg)
	}

	v, _ := b.Board()
	if len(v.Backlog) != 4 {
		t.Errorf("backlog len after :new = %d, want 4", len(v.Backlog))
	}
}

func TestCommandModeNewWithQuotedTitle(t *testing.T) {
	b := setupBoard(t)
	m, _ := newModel(b)

	updated, _ := m.Update(runeKey(':'))
	m = updated.(*Model)
	m = typeString(t, m, `new "round trip"`)
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal(":new produced no command")
	}
	if _, ok := cmd().(reloadedMsg); !ok {
		t.Fatal(":new didn't return a reloadedMsg")
	}

	v, _ := b.Board()
	found := false
	for _, e := range v.Backlog {
		if e.Title == "round trip" {
			found = true
			break
		}
	}
	if !found {
		titles := []string{}
		for _, e := range v.Backlog {
			titles = append(titles, e.Title)
		}
		t.Errorf(`:new "round trip" did not produce card titled "round trip"; backlog titles: %v`, titles)
	}
}

func TestSearchAcceptsPasteMsg(t *testing.T) {
	b := setupBoard(t)
	m, _ := newModel(b)
	v, _ := b.Board()
	updated, _ := m.Update(reloadedMsg{view: v})
	m = updated.(*Model)

	updated, _ = m.Update(runeKey('/'))
	m = updated.(*Model)
	if m.input != modeSearch {
		t.Fatalf("input = %v, want search", m.input)
	}

	pasted := "gämmå" // multi-byte to exercise unicode path
	updated, _ = m.Update(tea.PasteMsg{Content: pasted})
	m = updated.(*Model)
	if got := m.search.Value(); !strings.Contains(got, "g") {
		t.Errorf("paste didn't reach textinput; value=%q", got)
	}
}
