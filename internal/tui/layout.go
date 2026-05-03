package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// navNaturalWidth is the width the nav column wants in horizontal
// mode. Matches the formatRow output plus a small margin.
const navNaturalWidth = 80

// renderSplit composes nav + preview using the requested splitMode.
// Auto picks horizontal when width >= autoSplitThreshold, vertical
// otherwise. Horizontal gives nav up to navNaturalWidth and the rest
// to preview. Vertical splits the body height in half.
func renderSplit(mode splitMode, width, height int, nav *boardModel, prev *previewModel) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	effective := mode
	if effective == splitAuto {
		if width >= autoSplitThreshold {
			effective = splitHorizontal
		} else {
			effective = splitVertical
		}
	}

	if effective == splitHorizontal {
		return renderHorizontal(width, height, nav, prev)
	}
	return renderVertical(width, height, nav, prev)
}

func renderHorizontal(width, height int, nav *boardModel, prev *previewModel) string {
	navW := navNaturalWidth
	if navW > width-21 {
		navW = width - 21
	}
	if navW < 20 {
		navW = 20
	}
	previewW := width - navW - 1

	gutter := " "
	left := nav.view(navW, height)
	right := prev.view(previewW, height)

	left = padPaneLines(left, navW, height)
	right = padPaneLines(right, previewW, height)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, gutter, right)
}

func renderVertical(width, height int, nav *boardModel, prev *previewModel) string {
	return renderStacked(width, height, nav, prev)
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
