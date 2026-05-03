package tui

import (
	"strings"
	"testing"
)

func TestPadLine(t *testing.T) {
	cases := []struct {
		name  string
		in    string
		width int
		want  string
	}{
		{"shorter pads with spaces", "abc", 5, "abc  "},
		{"exact returns unchanged", "abc", 3, "abc"},
		{"longer truncates", "abcdef", 3, "abc"},
		{"ANSI not counted", "\x1b[31mabc\x1b[0m", 5, "\x1b[31mabc\x1b[0m  "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := padLine(tc.in, tc.width)
			if got != tc.want {
				t.Errorf("padLine(%q, %d) = %q, want %q", tc.in, tc.width, got, tc.want)
			}
		})
	}
}

func TestPadPaneLinesProducesRectangle(t *testing.T) {
	out := padPaneLines("foo\nbar baz", 6, 4)
	lines := strings.Split(out, "\n")
	if len(lines) != 4 {
		t.Errorf("got %d lines, want 4", len(lines))
	}
	for i, l := range lines {
		if len([]rune(l)) != 6 {
			t.Errorf("line[%d] = %q (len %d), want width 6", i, l, len([]rune(l)))
		}
	}
}

func TestRenderSplitAutoWidePicksHorizontal(t *testing.T) {
	b := setupBoard(t)
	m, _ := newModel(b)
	v, _ := b.Board()
	updated, _ := m.Update(reloadedMsg{view: v})
	m = updated.(*Model)

	out := renderSplit(splitAuto, 200, 20, &m.board_, &m.preview)
	firstLine := strings.SplitN(out, "\n", 2)[0]
	if !strings.Contains(firstLine, "ACTIVE") || !strings.Contains(firstLine, "#0001") {
		t.Errorf("auto+wide didn't go horizontal; first line = %q", firstLine)
	}
}

func TestRenderSplitAutoNarrowPicksVertical(t *testing.T) {
	b := setupBoard(t)
	m, _ := newModel(b)
	v, _ := b.Board()
	updated, _ := m.Update(reloadedMsg{view: v})
	m = updated.(*Model)

	const termWidth = 50
	out := renderSplit(splitAuto, termWidth, 20, &m.board_, &m.preview)
	if !strings.Contains(out, strings.Repeat("─", termWidth)) {
		t.Errorf("auto+narrow didn't go vertical: %q", out)
	}
}

func TestRenderSplitForcedHorizontalOnNarrowTerminal(t *testing.T) {
	b := setupBoard(t)
	m, _ := newModel(b)
	v, _ := b.Board()
	updated, _ := m.Update(reloadedMsg{view: v})
	m = updated.(*Model)

	out := renderSplit(splitHorizontal, 80, 20, &m.board_, &m.preview)
	firstLine := strings.SplitN(out, "\n", 2)[0]
	if !strings.Contains(firstLine, "ACTIVE") || !strings.Contains(firstLine, "#0001") {
		t.Errorf("forced horizontal didn't side-by-side; first line = %q", firstLine)
	}
}

func TestRenderSplitForcedVerticalOnWideTerminal(t *testing.T) {
	b := setupBoard(t)
	m, _ := newModel(b)
	v, _ := b.Board()
	updated, _ := m.Update(reloadedMsg{view: v})
	m = updated.(*Model)

	out := renderSplit(splitVertical, 200, 20, &m.board_, &m.preview)
	if !strings.Contains(out, strings.Repeat("─", 200)) {
		t.Error("forced vertical didn't stack")
	}
}

func TestSplitModeNextCycles(t *testing.T) {
	if got := splitAuto.next(); got != splitHorizontal {
		t.Errorf("auto.next = %v, want horizontal", got)
	}
	if got := splitHorizontal.next(); got != splitVertical {
		t.Errorf("horizontal.next = %v, want vertical", got)
	}
	if got := splitVertical.next(); got != splitAuto {
		t.Errorf("vertical.next = %v, want auto", got)
	}
}
