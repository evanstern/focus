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

func TestRenderSplitWideUsesHorizontal(t *testing.T) {
	b := setupBoard(t)
	m, _ := newModel(b)
	v, _ := b.Board()
	updated, _ := m.Update(reloadedMsg{view: v})
	m = updated.(*Model)

	out := renderSplit(splitWidthThreshold+10, 20, &m.board_, &m.preview)
	// Side-by-side: the first line should contain the nav header
	// AND the preview header (#0001 title) joined horizontally.
	firstLine := strings.SplitN(out, "\n", 2)[0]
	if !strings.Contains(firstLine, "ACTIVE") || !strings.Contains(firstLine, "#0001") {
		t.Errorf("wide layout didn't side-by-side; first line = %q", firstLine)
	}
}

func TestRenderSplitNarrowStacks(t *testing.T) {
	b := setupBoard(t)
	m, _ := newModel(b)
	v, _ := b.Board()
	updated, _ := m.Update(reloadedMsg{view: v})
	m = updated.(*Model)

	out := renderSplit(splitWidthThreshold-30, 20, &m.board_, &m.preview)
	// Stacked: the separator row of ─ runs should appear between
	// nav and preview.
	if !strings.Contains(out, strings.Repeat("─", splitWidthThreshold-30)) {
		t.Errorf("narrow layout didn't draw separator: %q", out)
	}
}
