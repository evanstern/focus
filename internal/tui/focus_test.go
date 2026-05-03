package tui

import (
	"strings"
	"testing"

	"github.com/evanstern/focus/internal/board"

	tea "github.com/charmbracelet/bubbletea"
)

func setupBoardForFocus(t *testing.T) *board.Board {
	t.Helper()
	root := t.TempDir()
	b, err := board.Init(root)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	longBody := strings.Repeat("paragraph line that gives the preview real height\n", 80)
	for _, title := range []string{"alpha", "beta", "gamma"} {
		c, _, err := b.NewCard(title, board.NewCardOpts{})
		if err != nil {
			t.Fatalf("NewCard %s: %v", title, err)
		}
		if err := b.SetBody(c.ID, longBody); err != nil {
			t.Fatalf("SetBody %s: %v", title, err)
		}
	}
	return b
}

func bootstrap(t *testing.T, b *board.Board) *Model {
	t.Helper()
	m, err := newModel(b)
	if err != nil {
		t.Fatal(err)
	}
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 200, Height: 40})
	m = updated.(*Model)
	v, err := b.Board()
	if err != nil {
		t.Fatal(err)
	}
	updated, _ = m.Update(reloadedMsg{view: v})
	m = updated.(*Model)
	_ = m.View()
	return m
}

func key(s string) tea.KeyMsg {
	switch s {
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
	case "ctrl+d":
		return tea.KeyMsg{Type: tea.KeyCtrlD}
	case "ctrl+u":
		return tea.KeyMsg{Type: tea.KeyCtrlU}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestTabCyclesFocusToPreview(t *testing.T) {
	b := setupBoardForFocus(t)
	m := bootstrap(t, b)

	if m.focused != focusNav {
		t.Fatalf("default focus = %v, want focusNav", m.focused)
	}

	updated, _ := m.Update(key("tab"))
	m = updated.(*Model)
	if m.focused != focusPreview {
		t.Errorf("after tab, focus = %v, want focusPreview", m.focused)
	}

	updated, _ = m.Update(key("tab"))
	m = updated.(*Model)
	if m.focused != focusNav {
		t.Errorf("after tab again, focus = %v, want focusNav", m.focused)
	}
}

func TestShiftTabCyclesFocus(t *testing.T) {
	b := setupBoardForFocus(t)
	m := bootstrap(t, b)

	updated, _ := m.Update(key("shift+tab"))
	m = updated.(*Model)
	if m.focused != focusPreview {
		t.Errorf("after shift+tab, focus = %v, want focusPreview", m.focused)
	}
}

func TestPreviewFocusedJScrollsViewport(t *testing.T) {
	b := setupBoardForFocus(t)
	m := bootstrap(t, b)

	beforeCursor := m.board_.cursor
	beforeOffset := m.preview.viewport.YOffset

	updated, _ := m.Update(key("tab"))
	m = updated.(*Model)
	updated, _ = m.Update(key("j"))
	m = updated.(*Model)

	if m.preview.viewport.YOffset != beforeOffset+1 {
		t.Errorf("YOffset = %d, want %d", m.preview.viewport.YOffset, beforeOffset+1)
	}
	if m.board_.cursor != beforeCursor {
		t.Errorf("nav cursor moved while preview was focused: was %d, now %d", beforeCursor, m.board_.cursor)
	}
}

func TestPreviewFocusedGgGoesToTopGGoesToBottom(t *testing.T) {
	b := setupBoardForFocus(t)
	m := bootstrap(t, b)

	updated, _ := m.Update(key("tab"))
	m = updated.(*Model)

	for i := 0; i < 5; i++ {
		updated, _ = m.Update(key("j"))
		m = updated.(*Model)
	}
	if m.preview.viewport.YOffset == 0 {
		t.Fatal("setup: expected non-zero YOffset before testing gg")
	}

	updated, _ = m.Update(key("g"))
	m = updated.(*Model)
	updated, _ = m.Update(key("g"))
	m = updated.(*Model)
	if m.preview.viewport.YOffset != 0 {
		t.Errorf("after gg, YOffset = %d, want 0", m.preview.viewport.YOffset)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = updated.(*Model)
	if !m.preview.viewport.AtBottom() {
		t.Errorf("after G, viewport not at bottom (YOffset=%d)", m.preview.viewport.YOffset)
	}
}

func TestSwitchingNavCursorResetsPreviewScroll(t *testing.T) {
	b := setupBoardForFocus(t)
	m := bootstrap(t, b)

	updated, _ := m.Update(key("tab"))
	m = updated.(*Model)
	for i := 0; i < 5; i++ {
		updated, _ = m.Update(key("j"))
		m = updated.(*Model)
	}
	if m.preview.viewport.YOffset == 0 {
		t.Fatal("setup: expected non-zero YOffset before tab back")
	}

	updated, _ = m.Update(key("tab"))
	m = updated.(*Model)
	if m.focused != focusNav {
		t.Fatalf("focus = %v, want focusNav", m.focused)
	}

	updated, _ = m.Update(key("j"))
	m = updated.(*Model)
	_ = m.View()

	if m.preview.viewport.YOffset != 0 {
		t.Errorf("after nav cursor moved to a new card, preview YOffset = %d, want 0",
			m.preview.viewport.YOffset)
	}
}

func TestStatusHintIncludesTabFocus(t *testing.T) {
	b := setupBoardForFocus(t)
	m := bootstrap(t, b)
	if !strings.Contains(m.statusContent(), "tab focus") {
		t.Errorf("status hint missing 'tab focus': %q", m.statusContent())
	}
}

func TestFilterCycleWorksRegardlessOfFocus(t *testing.T) {
	b := setupBoardForFocus(t)
	m := bootstrap(t, b)

	updated, _ := m.Update(key("tab"))
	m = updated.(*Model)
	if m.focused != focusPreview {
		t.Fatalf("focus = %v, want focusPreview", m.focused)
	}

	_, cmd := m.Update(key("l"))
	if cmd == nil {
		t.Error("'l' produced no command while preview focused; filter cycle should still work")
	}
}

func TestNavFocusedJStillMovesCursor(t *testing.T) {
	b := setupBoardForFocus(t)
	m := bootstrap(t, b)

	first := m.board_.selectedCard()
	if first == nil {
		t.Fatal("no card selected")
	}
	updated, _ := m.Update(key("j"))
	m = updated.(*Model)
	got := m.board_.selectedCard()
	if got == nil || got.ID == first.ID {
		t.Errorf("j with nav focus should move cursor: was %v, now %v", first, got)
	}
}
