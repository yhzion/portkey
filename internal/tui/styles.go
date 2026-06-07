package tui

import "github.com/charmbracelet/lipgloss"

// Palette: one identity color, three semantic, two neutrals.
// No other hex values anywhere in this package.

const (
	colorPrimary  = "#7D56F7" // brand purple — title, cursor, selected
	colorDanger   = "#F87171" // red — errors, destructive actions
	colorPositive = "#4ADE80" // green — add, success
	colorBright   = "#E2E2E2" // readable text
	colorDim      = "#6B7280" // help, secondary info
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colorPrimary))

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorPrimary)).
			Bold(true)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorBright)).
			Background(lipgloss.Color(colorPrimary))

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorBright))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorDim))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorDim)).
			MarginTop(1)

	indexStyle = lipgloss.NewStyle().
			Width(4).
			Foreground(lipgloss.Color(colorDim))

	nameStyle = lipgloss.NewStyle().
			Width(20).
			Bold(true)

	addItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorPositive))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorDanger))
)
