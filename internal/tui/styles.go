package tui

import "github.com/charmbracelet/lipgloss"

// styleSet is the package-wide palette. Fixed colors (not Adaptive)
// because lipgloss's HasDarkBackground query is unreliable in
// tmux/screen — the same query that breaks termenv's profile
// detection. We pick colors that read on both light and dark
// terminals, biased toward dark since that's the common case.
var styles = struct {
	header        lipgloss.Style
	cursor        lipgloss.Style
	priority      [4]lipgloss.Style
	dim           lipgloss.Style
	statusBg      lipgloss.Style
	border        lipgloss.Style
	borderFocused lipgloss.Style
}{
	header: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7aa2f7")),
	cursor: lipgloss.NewStyle().Background(lipgloss.Color("#3b4261")).Foreground(lipgloss.Color("#ffffff")),
	priority: [4]lipgloss.Style{
		lipgloss.NewStyle().Foreground(lipgloss.Color("#f7768e")).Bold(true),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#e0af68")),
		lipgloss.NewStyle(),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#666")),
	},
	dim:           lipgloss.NewStyle().Foreground(lipgloss.Color("#888")),
	statusBg:      lipgloss.NewStyle().Foreground(lipgloss.Color("#aaa")),
	border:        lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#444")),
	borderFocused: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#e0af68")),
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
