package tui

import "github.com/charmbracelet/lipgloss"

// styleSet is the package-wide palette. Defined once so colors and
// adaptive styling stay consistent across nav, preview, status bar.
//
// Adaptive colors mean the same logical role (e.g. "section header")
// renders differently on light vs dark terminals. Lipgloss handles
// the detection.
var styles = struct {
	header   lipgloss.Style
	cursor   lipgloss.Style
	priority [4]lipgloss.Style
	dim      lipgloss.Style
	statusBg lipgloss.Style
}{
	header: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#444", Dark: "#bbb"}),
	cursor: lipgloss.NewStyle().Background(lipgloss.AdaptiveColor{Light: "#cde", Dark: "#235"}).Foreground(lipgloss.AdaptiveColor{Light: "#000", Dark: "#fff"}),
	priority: [4]lipgloss.Style{
		lipgloss.NewStyle().Foreground(lipgloss.Color("#e06060")).Bold(true),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#e0c060")),
		lipgloss.NewStyle(),
		lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#888", Dark: "#666"}),
	},
	dim:      lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#888", Dark: "#666"}),
	statusBg: lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#666", Dark: "#aaa"}),
}

// priorityStyle returns the lipgloss style to apply to a priority
// cell. p0 -> red, p1 -> yellow, p2 -> default, p3 -> dim.
func priorityStyle(p string) lipgloss.Style {
	switch p {
	case "p0":
		return styles.priority[0]
	case "p1":
		return styles.priority[1]
	case "p3":
		return styles.priority[3]
	}
	return styles.priority[2]
}
