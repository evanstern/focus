package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// renderSplit composes nav + preview into a single screen. When the
// terminal is at least splitWidthThreshold cols wide we place them
// side-by-side; below that we stack nav on top of preview.
//
// The split point is fixed at 50% on stacked mode (nav gets the top
// half) and at the nav's natural width on side-by-side mode (which
// caps at half the terminal width — gives the preview room to
// breathe on very wide terminals).
func renderSplit(width, height int, nav *boardModel, prev *previewModel) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	if width >= splitWidthThreshold {
		return renderSideBySide(width, height, nav, prev)
	}
	return renderStacked(width, height, nav, prev)
}

func renderSideBySide(width, height int, nav *boardModel, prev *previewModel) string {
	navWidth := navColumnWidth(width)
	previewWidth := width - navWidth - 1
	gutter := " "

	left := nav.view(navWidth, height)
	right := prev.view(previewWidth, height)

	left = padPaneLines(left, navWidth, height)
	right = padPaneLines(right, previewWidth, height)

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

// navColumnWidth picks how wide the nav column should be in
// side-by-side mode. We give nav 80 cols (its natural row width)
// when the terminal is wide enough; otherwise we split 50/50.
func navColumnWidth(termWidth int) int {
	const natural = 80
	if termWidth >= natural*2 {
		return natural
	}
	return termWidth / 2
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
