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
		b.WriteString(dimStyle.Render("⬆ Update available: " + m.updateTag))
	}
	b.WriteString("\n\n")

	if len(m.config.Hosts) == 0 {
		b.WriteString(m.renderAddItem(true))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("No hosts registered yet."))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("↑/↓ move · enter select · a add · q quit"))
		return b.String()
	}

	for i, host := range m.config.Hosts {
		b.WriteString(m.renderHostItem(i, host, i == m.selected))
	}

	b.WriteString(m.renderAddItem(m.selected == len(m.config.Hosts)))

	b.WriteString("\n")
	b.WriteString(
		helpStyle.Render(
			"↑/↓ move · enter/space select · 1-9 quick select · a add · e edit · d delete · u update · q quit",
		),
	)

	return b.String()
}

func (m *model) renderHostItem(index int, host config.Host, selected bool) string {
	connInfo := fmt.Sprintf("%s@%s", host.Username, host.Host)
	if host.Port != 22 {
		connInfo = fmt.Sprintf("%s:%d", connInfo, host.Port)
	}

	line := fmt.Sprintf("%s %s %s",
		indexStyle.Render(fmt.Sprintf("%d.", index+1)),
		nameStyle.Render(host.DisplayName),
		dimStyle.Render(connInfo),
	)

	if selected {
		return cursorStyle.Render("▸ ") + selectedStyle.Render(line) + "\n"
	}
	return "  " + normalStyle.Render(line) + "\n"
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
	b.WriteString(errorStyle.Render(fmt.Sprintf("Delete \"%s\"?", host.DisplayName)))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("[y] yes / [n] no"))
	return b.String()
}

func (m *model) renderUpdateConfirm() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(normalStyle.Render(fmt.Sprintf("Update portkey %s → %s?", m.Version, m.updateTag)))
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
