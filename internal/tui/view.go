package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yhzion/portkey/internal/config"
)

func (m *model) View() string {
	switch m.screen {
	case screenHostList:
		return m.renderHostList()
	case screenAddHost:
		return m.renderFormScreen("Add Host")
	case screenEditHost:
		return m.renderFormScreen("Edit Host")
	case screenDeleteConfirm:
		return m.renderDeleteConfirm()
	case screenUpdateConfirm:
		return m.renderUpdateConfirm()
	case screenError:
		return m.renderError()
	}
	return ""
}

func (m *model) renderHostList() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Portkey"))
	if m.updateTag != "" {
		b.WriteString(" ")
		b.WriteString(accentStyle.Render("✨ " + m.updateTag + " available — press u to update"))
	}
	b.WriteString("\n\n")

	// Search bar.
	if m.search.Active {
		b.WriteString(searchBarStyle.Render("/> " + m.search.Query + "█"))
		b.WriteString("\n\n")
	}

	// Use filtered hosts when search is active.
	visible := m.search.Indices()
	if visible == nil {
		visible = make([]int, len(m.config.Hosts))
		for i := range m.config.Hosts {
			visible[i] = i
		}
	}
	matchMap := m.search.MatchMap() // for highlight positions

	if len(m.config.Hosts) == 0 {
		b.WriteString(m.renderAddItem(true))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("No hosts registered yet."))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("↑/↓ move · enter select · / search · a add · q quit"))
		return b.String()
	}

	// Empty search results.
	if m.search.Active && len(visible) == 0 {
		b.WriteString(dimStyle.Render("No matches found."))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("esc clear search · backspace delete · q quit"))
		return b.String()
	}

	for displayIdx, hostIdx := range visible {
		host := m.config.Hosts[hostIdx]
		selected := displayIdx == m.selected
		var positions []int
		if matchMap != nil {
			if mr, ok := matchMap[hostIdx]; ok {
				positions = mr.Positions
			}
		}
		b.WriteString(m.renderHostItem(displayIdx, host, selected, positions))
	}

	b.WriteString(m.renderAddItem(m.selected == len(visible)))

	b.WriteString("\n")
	if m.search.Active {
		b.WriteString(
			helpStyle.Render(
				"type to filter · ↑/↓ move · enter/space select · 1-9 quick · esc clear · q quit",
			),
		)
	} else {
		b.WriteString(
			helpStyle.Render(
				"↑/↓ move · enter/space select · 1-9 quick select · / search · a add · e edit · d delete · u update · q quit",
			),
		)
	}

	return b.String()
}

func (m *model) renderHostItem(
	index int,
	host config.Host,
	selected bool,
	matchPositions []int,
) string {
	connInfo := fmt.Sprintf("%s@%s", host.Username, host.Host)
	if host.Port != 22 {
		connInfo = fmt.Sprintf("%s:%d", connInfo, host.Port)
	}

	nameStr := host.Name
	if len(matchPositions) > 0 {
		nameStr = highlightMatched(host.Name, matchPositions)
	}

	line := fmt.Sprintf(
		"%s %s %s",
		indexStyle.Render(fmt.Sprintf("%d.", index+1)),
		nameStyle.Render(nameStr),
		dimStyle.Render(connInfo),
	)

	if selected {
		return cursorStyle.Render("▸ ") + selectedStyle.Render(line) + "\n"
	}
	return "  " + normalStyle.Render(line) + "\n"
}

// highlightMatched renders a string with matched character positions highlighted.
func highlightMatched(s string, positions []int) string {
	runes := []rune(s)
	posSet := make(map[int]bool, len(positions))
	for _, p := range positions {
		if p < len(runes) {
			posSet[p] = true
		}
	}

	var b strings.Builder
	for i, r := range runes {
		if posSet[i] {
			b.WriteString(matchStyle.Render(string(r)))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (m *model) renderAddItem(selected bool) string {
	label := "+ Add new host"
	if selected {
		return cursorStyle.Render("▸ ") + selectedStyle.Render("  "+label) + "\n"
	}
	return "  " + addItemStyle.Render(label) + "\n"
}

func (m *model) renderFormScreen(label string) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Portkey"))
	b.WriteString(" ")
	b.WriteString(dimStyle.Render("·"))
	b.WriteString(" ")
	b.WriteString(normalStyle.Render(label))
	b.WriteString("\n\n")
	b.WriteString(m.form.View())
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("esc cancel"))
	return b.String()
}

func (m *model) renderDeleteConfirm() string {
	host := m.config.Hosts[m.editIndex]

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(errorStyle.Render(fmt.Sprintf("Delete \"%s\"?", host.Name)))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("[y] yes / [n] no"))
	return b.String()
}

func (m *model) renderUpdateConfirm() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(accentStyle.Render(fmt.Sprintf("✨ New version (%s) detected. Update now?", m.updateTag)))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("[y] yes / [n] no"))
	return b.String()
}

func (m *model) renderError() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(errorStyle.Render("Error: " + m.errMsg))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("Press any key to return to host list."))
	return b.String()
}

var _ tea.Model = (*model)(nil)
