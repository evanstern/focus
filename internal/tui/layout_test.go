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

// firstLineRowCount counts the number of border-top corner glyphs
// in the first row. Two corners means side-by-side (two panes
// horizontal), one corner means stacked.
func topRowCornerCount(out string) int {
	first := strings.SplitN(out, "\n", 2)[0]
	return strings.Count(first, "╭")
}

func TestRenderSplitAutoWidePicksHorizontal(t *testing.T) {
	b := setupBoard(t)
	m, _ := newModel(b)
	v, _ := b.Board()
	updated, _ := m.Update(reloadedMsg{view: v})
	m = updated.(*Model)

	out := renderSplit(splitAuto, 200, 20, focusNav, &m.board_, &m.preview)
	if got := topRowCornerCount(out); got != 2 {
		t.Errorf("auto+wide top-row corners = %d, want 2 (horizontal): %q",
			got, strings.SplitN(out, "\n", 2)[0])
	}
}

func TestRenderSplitAutoNarrowPicksVertical(t *testing.T) {
	b := setupBoard(t)
	m, _ := newModel(b)
	v, _ := b.Board()
	updated, _ := m.Update(reloadedMsg{view: v})
	m = updated.(*Model)

	out := renderSplit(splitAuto, 50, 20, focusNav, &m.board_, &m.preview)
	if got := topRowCornerCount(out); got != 1 {
		t.Errorf("auto+narrow top-row corners = %d, want 1 (vertical)", got)
	}
}

func TestRenderSplitForcedHorizontalOnNarrowTerminal(t *testing.T) {
	b := setupBoard(t)
	m, _ := newModel(b)
	v, _ := b.Board()
	updated, _ := m.Update(reloadedMsg{view: v})
	m = updated.(*Model)

	out := renderSplit(splitHorizontal, 80, 20, focusNav, &m.board_, &m.preview)
	if got := topRowCornerCount(out); got != 2 {
		t.Errorf("forced horizontal top-row corners = %d, want 2", got)
	}
}

func TestRenderSplitForcedVerticalOnWideTerminal(t *testing.T) {
	b := setupBoard(t)
	m, _ := newModel(b)
	v, _ := b.Board()
	updated, _ := m.Update(reloadedMsg{view: v})
	m = updated.(*Model)

	out := renderSplit(splitVertical, 200, 20, focusNav, &m.board_, &m.preview)
	if got := topRowCornerCount(out); got != 1 {
		t.Errorf("forced vertical top-row corners = %d, want 1 (stacked)", got)
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
