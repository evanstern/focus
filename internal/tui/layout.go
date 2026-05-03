package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// navNaturalWidth is the width the nav column wants when there's
// room. It matches the formatRow output (#0001 + 40-char title +
// project + priority + owner with separators) plus a small margin.
const navNaturalWidth = 80

// renderSplit composes nav + preview into a single screen, sizing
// each pane to its content. Side-by-side when the terminal can fit
// nav at its natural width plus the preview's content width;
// otherwise stacked so the preview gets the full terminal width.
//
// "Content-aware" means the preview pane's width is driven by the
// rendered card's actual longest line, not a fixed 50/50 split. A
// short card gets a narrow preview pane and the nav grows to fill
// the rest; a wide card forces the layout to stack so nothing gets
// truncated unnecessarily.
func renderSplit(width, height int, nav *boardModel, prev *previewModel) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	const navMinWidth = 50
	contentW := prev.contentWidth()
	if contentW < 1 {
		contentW = 1
	}

	if navNaturalWidth+1+contentW <= width {
		return renderSideBySide(width, height, nav, prev, navNaturalWidth, contentW)
	}

	if navMinWidth+1+contentW <= width {
		navW := width - 1 - contentW
		if navW > navNaturalWidth {
			navW = navNaturalWidth
		}
		return renderSideBySide(width, height, nav, prev, navW, width-navW-1)
	}

	return renderStacked(width, height, nav, prev)
}

func renderSideBySide(width, height int, nav *boardModel, prev *previewModel, navW, previewW int) string {
	gutter := " "

	left := nav.view(navW, height)
	right := prev.view(previewW, height)

	left = padPaneLines(left, navW, height)
	right = padPaneLines(right, previewW, height)

	_ = width
	return lipgloss.JoinHorizontal(lipgloss.Top, left, gutter, right)
}

func renderStacked(width, height int, nav *boardModel, prev *previewModel) string {
	navHeight := height / 2
	previewHeight := height - navHeight - 1
	if navHeight < 3 {
		navHeight = 3
	}
	if previewHeight < 1 {
		previewHeight = 1
	}

	top := nav.view(width, navHeight)
	bottom := prev.view(width, previewHeight)

	top = padPaneLines(top, width, navHeight)
	bottom = padPaneLines(bottom, width, previewHeight)

	separator := strings.Repeat("─", width)
	return lipgloss.JoinVertical(lipgloss.Left, top, separator, bottom)
}

// padPaneLines pads each line of s to width and pads the line count
// to height, so the lipgloss join produces a clean rectangle. Lines
// longer than width are truncated.
func padPaneLines(s string, width, height int) string {
	if width <= 0 {
		width = 1
	}
	if height <= 0 {
		height = 1
	}
	lines := strings.Split(s, "\n")
	out := make([]string, 0, height)
	for i := 0; i < height; i++ {
		var line string
		if i < len(lines) {
			line = lines[i]
		}
		out = append(out, padLine(line, width))
	}
	return strings.Join(out, "\n")
}

// padLine right-pads or truncates a single line to exactly width
// terminal columns. Uses ansi.StringWidth so ANSI escape sequences
// emitted by glamour and lipgloss don't throw off the measurement.
func padLine(s string, width int) string {
	w := ansi.StringWidth(s)
	if w == width {
		return s
	}
	if w < width {
		return s + strings.Repeat(" ", width-w)
	}
	return ansi.Truncate(s, width, "")
}
