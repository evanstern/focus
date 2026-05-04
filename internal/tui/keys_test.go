package tui

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

func TestKeyMapMatchesEachBinding(t *testing.T) {
	km := DefaultKeyMap()
	cases := []struct {
		name string
		msg  tea.KeyPressMsg
		want key.Binding
	}{
		{"tab → FocusNext", tea.KeyPressMsg{Code: tea.KeyTab}, km.FocusNext},
		{"shift+tab → FocusPrev", tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}, km.FocusPrev},
		{"k → Up", tea.KeyPressMsg{Code: 'k', Text: "k"}, km.Up},
		{"up → Up", tea.KeyPressMsg{Code: tea.KeyUp}, km.Up},
		{"j → Down", tea.KeyPressMsg{Code: 'j', Text: "j"}, km.Down},
		{"down → Down", tea.KeyPressMsg{Code: tea.KeyDown}, km.Down},
		{"g → Top", tea.KeyPressMsg{Code: 'g', Text: "g"}, km.Top},
		{"G → Bottom", tea.KeyPressMsg{Code: 'G', Text: "G"}, km.Bottom},
		{"ctrl+d → JumpDown", tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl}, km.JumpDown},
		{"ctrl+u → JumpUp", tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl}, km.JumpUp},
		{"ctrl+f → ScrollPgDown", tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl}, km.ScrollPgDown},
		{"ctrl+b → ScrollPgUp", tea.KeyPressMsg{Code: 'b', Mod: tea.ModCtrl}, km.ScrollPgUp},
		{"l → FilterNext", tea.KeyPressMsg{Code: 'l', Text: "l"}, km.FilterNext},
		{"h → FilterPrev", tea.KeyPressMsg{Code: 'h', Text: "h"}, km.FilterPrev},
		{"s → LayoutCycle", tea.KeyPressMsg{Code: 's', Text: "s"}, km.LayoutCycle},
		{"enter → Edit", tea.KeyPressMsg{Code: tea.KeyEnter}, km.Edit},
		{"e → Edit", tea.KeyPressMsg{Code: 'e', Text: "e"}, km.Edit},
		{"o → Edit", tea.KeyPressMsg{Code: 'o', Text: "o"}, km.Edit},
		{"a → Activate", tea.KeyPressMsg{Code: 'a', Text: "a"}, km.Activate},
		{"p → Park", tea.KeyPressMsg{Code: 'p', Text: "p"}, km.Park},
		{"d → Done", tea.KeyPressMsg{Code: 'd', Text: "d"}, km.Done},
		{"K → Kill", tea.KeyPressMsg{Code: 'K', Text: "K"}, km.Kill},
		{"r → Revive", tea.KeyPressMsg{Code: 'r', Text: "r"}, km.Revive},
		{"/ → Search", tea.KeyPressMsg{Code: '/', Text: "/"}, km.Search},
		{": → Command", tea.KeyPressMsg{Code: ':', Text: ":"}, km.Command},
		{"? → Help", tea.KeyPressMsg{Code: '?', Text: "?"}, km.Help},
		{"q → Quit", tea.KeyPressMsg{Code: 'q', Text: "q"}, km.Quit},
		{"ctrl+c → Quit", tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}, km.Quit},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !key.Matches(tc.msg, tc.want) {
				t.Errorf("key.Matches(%q, %v) = false, want true",
					tc.msg.String(), tc.want.Help())
			}
		})
	}
}

func TestHelpViewRendersFromKeyMap(t *testing.T) {
	b := setupBoard(t)
	m, err := newModel(b)
	if err != nil {
		t.Fatal(err)
	}
	m.help.ShowAll = true
	out := m.help.View(m.keys)
	if out == "" {
		t.Fatal("help.View returned empty string")
	}
	if !strings.Contains(out, "tab") {
		t.Errorf("help view missing 'tab' key: %q", out)
	}
	if !strings.Contains(out, "focus next pane") {
		t.Errorf("help view missing FocusNext description: %q", out)
	}
}
