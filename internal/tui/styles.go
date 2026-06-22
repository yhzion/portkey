package tui

import "github.com/charmbracelet/lipgloss"

// Palette: one identity color, one foreground variant, three semantic, two neutrals.
// All foreground-on-dark combinations meet WCAG 2.1 AA (≥4.5:1) across common terminals.
// No other hex values anywhere in this package.

const (
	colorPrimary  = "#440F8D" // dark purple — selected row background
	colorFg       = "#BA82F4" // light purple — title, cursor, search (fg on dark)
	colorDanger   = "#F87171" // red — errors, destructive actions
	colorPositive = "#4ADE80" // green — add, success
	colorBright   = "#FFFFFF" // white — primary text, selected row fg
	colorDim      = "#9CA3AF" // Tailwind gray-400 — help, indexes, secondary info
	colorAccent   = "#FBBF24" // gold — update notifications, search match
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colorFg))

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorFg)).
			Bold(true)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorBright)).
			Background(lipgloss.Color(colorPrimary))

	// nameStyle has no explicit foreground by default, so an unchecked (unknown)
	// host's name inherits its parent's color (colorBright inside selectedStyle
	// or normalStyle). For online/offline hosts, styledName overrides the
	// foreground (green / dim), which wins because the name is rendered to a
	// finished ANSI string before being wrapped by the row style.

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

	// nameStyle's width is applied per-render via nameColumnWidth (see view.go),
	// so the column fits the longest visible name instead of a fixed 20.
	nameStyle = lipgloss.NewStyle().
			Bold(true)

	addItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorPositive))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorDanger))

	accentStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorAccent)).
			Bold(true)

	searchBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorFg)).
			Bold(true)

	matchStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorAccent)).
			Bold(true)
)

// styledName returns the name-cell style for a host, tinted by health status.
func styledName(status healthStatus, width int) lipgloss.Style {
	s := nameStyle.Width(width)
	if c := nameColor(status); c != "" {
		s = s.Foreground(lipgloss.Color(c))
	}
	return s
}
