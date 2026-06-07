package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F7")).
			MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#3C3C6E"))

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F7")).
			Bold(true)

	addItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04DE5A"))

	selectedAddStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#3C3C6E"))

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CCCCCC"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			MarginTop(1)

	emptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			MarginBottom(1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5555"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04DE5A"))

	deleteConfirmStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFAA00"))

	indexStyle = lipgloss.NewStyle().
			Width(4).
			Foreground(lipgloss.Color("#888888"))

	nameStyle = lipgloss.NewStyle().
			Width(20).
			Bold(true)

	connStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#AAAAAA"))
)
