package tui

import (
	"strings"
	"testing"

	"github.com/evanstern/focus/internal/board"

	tea "github.com/charmbracelet/bubbletea"
)

// setupBoard creates a tempdir board with a few cards and returns
// the resolved Board. The TUI tests run the Update function directly
// — no real terminal involved. teatest exists but the Bubble Tea
// docs flag it as experimental; for our needs (unit-testing model
// behavior) calling Update with synthetic messages is sufficient.
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

func TestModelBoardCursorMovement(t *testing.T) {
	b := setupBoard(t)
	m, err := newModel(b)
	if err != nil {
		t.Fatal(err)
	}
	// Trigger initial reload synchronously.
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

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(*Model)
	second := m.board_.selectedCard()
	if second == nil || second.ID == first.ID {
		t.Errorf("j didn't move cursor; first=%v second=%v", first, second)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
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

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
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

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter produced no command")
	}
	// Don't actually invoke the editor in tests — just confirm a
	// command was produced. tea.ExecProcess returns a tea.Cmd that
	// isn't safe to call here (it'd try to attach the test process to
	// a TTY).
}

func TestModelHelpToggle(t *testing.T) {
	b := setupBoard(t)
	m, _ := newModel(b)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = updated.(*Model)
	if m.view != viewHelp {
		t.Errorf("view after ? = %v, want help", m.view)
	}
	if !strings.Contains(m.help.view(), "MOVEMENT") {
		t.Error("help text missing MOVEMENT section")
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
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

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = updated.(*Model)
	if m.input != modeSearch {
		t.Fatalf("input = %v, want search", m.input)
	}
	for _, r := range "gamma" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(*Model)
	}
	if got := m.board_.selectedCard(); got == nil || got.Title != "gamma" {
		t.Errorf("live search didn't jump to gamma; got %+v", got)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
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

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = updated.(*Model)
	for _, r := range "0002" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(*Model)
	}
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

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = updated.(*Model)
	for _, r := range "gamma" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(*Model)
	}
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

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
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

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = updated.(*Model)
	for _, r := range "zzzzz" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(*Model)
	}
	if m.status != "no matches" {
		t.Errorf("expected 'no matches' status; got %q", m.status)
	}
	for i := 0; i < 5; i++ {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
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
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		m = updated.(*Model)
	}
	beforeID := m.board_.selectedCard().ID

	updated, _ = m.Update(editFinishedMsg{id: beforeID})
	m = updated.(*Model)
	cmd := m.Init
	_ = cmd

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
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	m = updated.(*Model)
	if m.input != modeCommand {
		t.Fatalf("input = %v, want command", m.input)
	}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updated.(*Model)
	updated, cmd2 := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(*Model)
	// Either the second or third call should return a Quit command.
	if cmd == nil && cmd2 == nil {
		t.Error("command :q didn't produce a Quit cmd")
	}
}

func TestCommandModeNewCreatesCard(t *testing.T) {
	b := setupBoard(t)
	m, _ := newModel(b)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	m = updated.(*Model)
	for _, r := range "new fresh card" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(*Model)
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
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
