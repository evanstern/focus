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

// borderChrome is the space borders consume around a pane: 2 columns
// (left + right border) and 2 rows (top + bottom border).
const borderChrome = 2

func renderHorizontal(width, height int, nav *boardModel, prev *previewModel) string {
	navOuter := navNaturalWidth + borderChrome
	if navOuter > width-(20+borderChrome) {
		navOuter = width - (20 + borderChrome)
	}
	if navOuter < 20+borderChrome {
		navOuter = 20 + borderChrome
	}
	previewOuter := width - navOuter

	innerHeight := height - borderChrome
	if innerHeight < 1 {
		innerHeight = 1
	}

	left := borderedPane(nav.view(navOuter-borderChrome, innerHeight), navOuter-borderChrome, innerHeight)
	right := borderedPane(prev.view(previewOuter-borderChrome, innerHeight), previewOuter-borderChrome, innerHeight)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func renderVertical(width, height int, nav *boardModel, prev *previewModel) string {
	return renderStacked(width, height, nav, prev)
}

func renderStacked(width, height int, nav *boardModel, prev *previewModel) string {
	navOuterH := height / 2
	previewOuterH := height - navOuterH
	if navOuterH < 3+borderChrome {
		navOuterH = 3 + borderChrome
	}
	if previewOuterH < 1+borderChrome {
		previewOuterH = 1 + borderChrome
	}

	innerW := width - borderChrome
	if innerW < 1 {
		innerW = 1
	}

	top := borderedPane(nav.view(innerW, navOuterH-borderChrome), innerW, navOuterH-borderChrome)
	bottom := borderedPane(prev.view(innerW, previewOuterH-borderChrome), innerW, previewOuterH-borderChrome)

	return lipgloss.JoinVertical(lipgloss.Left, top, bottom)
}

// borderedPane wraps content in a rounded gray border, padding the
// content to exactly innerW × innerH first so the bordered output is
// a clean rectangle. Without the explicit pad, lipgloss's border
// would size itself to the longest line and cause the layout to
// shift on every cursor move.
func borderedPane(content string, innerW, innerH int) string {
	padded := padPaneLines(content, innerW, innerH)
	return styles.border.Render(padded)
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
