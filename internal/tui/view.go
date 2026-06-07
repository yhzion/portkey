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
		return m.renderAddHost()
	case screenEditHost:
		return m.renderEditHost()
	case screenDeleteConfirm:
		return m.renderDeleteConfirm()
	case screenError:
		return m.renderError()
	}
	return ""
}

func (m *model) renderHostList() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Portkey"))
	b.WriteString("\n\n")

	if len(m.config.Hosts) == 0 {
		b.WriteString(m.renderAddItem(0, true))
		b.WriteString("\n")
		b.WriteString(emptyStyle.Render("No hosts registered yet."))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("↑/↓ move · enter select · a add · q quit"))
		return b.String()
	}

	for i, host := range m.config.Hosts {
		isSelected := i == m.selected
		b.WriteString(m.renderHostItem(i, host, isSelected))
	}

	b.WriteString(m.renderAddItem(len(m.config.Hosts), m.selected == len(m.config.Hosts)))

	b.WriteString("\n")
	b.WriteString(m.renderHelp())

	return b.String()
}

func (m *model) renderHostItem(index int, host config.Host, selected bool) string {
	indexStr := fmt.Sprintf("%d.", index+1)
	connInfo := fmt.Sprintf("%s@%s", host.Username, host.Host)
	if host.Port != 22 {
		connInfo = fmt.Sprintf("%s:%d", connInfo, host.Port)
	}

	line := fmt.Sprintf("%s %s %s",
		indexStyle.Render(indexStr),
		nameStyle.Render(host.DisplayName),
		connStyle.Render(connInfo),
	)

	if selected {
		return cursorStyle.Render("▸ ") + selectedStyle.Render(line) + "\n"
	}
	return "  " + normalStyle.Render(line) + "\n"
}

func (m *model) renderAddItem(index int, selected bool) string {
	label := "+ Add new host"
	if selected {
		return cursorStyle.Render("▸ ") + selectedAddStyle.Render("  "+label) + "\n"
	}
	return "  " + addItemStyle.Render(label) + "\n"
}

func (m *model) renderHelp() string {
	return helpStyle.Render("↑/↓ move · enter/space select · 1-9 quick select · a add · e edit · d delete · q quit")
}

func (m *model) renderAddHost() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Portkey"))
	b.WriteString(" — ")
	b.WriteString(successStyle.Render("Add Host"))
	b.WriteString("\n\n")

	view := m.form.View()
	b.WriteString(view)

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("esc cancel"))
	return b.String()
}

func (m *model) renderEditHost() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Portkey"))
	b.WriteString(" — ")
	b.WriteString(successStyle.Render("Edit Host"))
	b.WriteString("\n\n")

	view := m.form.View()
	b.WriteString(view)

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("esc cancel"))
	return b.String()
}

func (m *model) renderDeleteConfirm() string {
	host := m.config.Hosts[m.editIndex]

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(deleteConfirmStyle.Render(fmt.Sprintf("Delete \"%s\"?", host.DisplayName)))
	b.WriteString("\n\n")
	b.WriteString(normalStyle.Render("[y] yes / [n] no"))
	return b.String()
}

func (m *model) renderError() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(errorStyle.Render("Error: " + m.errMsg))
	b.WriteString("\n\n")
	b.WriteString(normalStyle.Render("Press any key to return to host list."))
	return b.String()
}

var _ tea.Model = (*model)(nil)
