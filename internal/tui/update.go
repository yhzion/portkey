package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/yhzion/portkey/internal/config"
	"github.com/yhzion/portkey/internal/ssh"
)

func sshRun(host config.Host) error {
	return ssh.Run(host)
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.form != nil {
			m.form = m.form.WithWidth(min(msg.Width-4, 80))
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case sshDoneMsg:
		if msg.err != nil {
			m.errMsg = fmt.Sprintf("SSH error: %s", msg.err.Error())
			m.screen = screenError
		} else {
			m.screen = screenHostList
		}
		return m, nil

	case errMsg:
		m.errMsg = msg.Error()
		m.screen = screenError
		return m, nil

	case nil:
		return m, nil
	}

	return m, nil
}

func (m *model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenHostList:
		return m.handleHostListKey(msg)
	case screenAddHost, screenEditHost:
		return m.handleFormKey(msg)
	case screenDeleteConfirm:
		return m.handleDeleteConfirmKey(msg)
	case screenError:
		return m.handleErrorKey(msg)
	}
	return m, nil
}

func (m *model) handleHostListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	totalItems := len(m.config.Hosts) + 1 // hosts + "Add new host"

	switch {
	case key.Matches(msg, m.keys.Up):
		if m.selected > 0 {
			m.selected--
		}
	case key.Matches(msg, m.keys.Down):
		if m.selected < totalItems-1 {
			m.selected++
		}
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case msg.String() == "a":
		return m, m.showAddScreen()
	case key.Matches(msg, m.keys.Edit):
		if m.selected < len(m.config.Hosts) {
			return m, m.showEditScreen(m.selected)
		}
	case key.Matches(msg, m.keys.Delete):
		if m.selected < len(m.config.Hosts) {
			m.showDeleteConfirm(m.selected)
		}
	case key.Matches(msg, m.keys.Enter), key.Matches(msg, m.keys.Space):
		return m.handleSelect()
	default:
		if len(msg.String()) == 1 && msg.String() >= "1" && msg.String() <= "9" {
			num := int(msg.String()[0] - '0')
			if num <= len(m.config.Hosts) {
				return m, m.connectHost(num - 1)
			}
		}
	}
	return m, nil
}

func (m *model) handleSelect() (tea.Model, tea.Cmd) {
	if m.selected == len(m.config.Hosts) {
		return m, m.showAddScreen()
	}
	if m.selected < len(m.config.Hosts) {
		return m, m.connectHost(m.selected)
	}
	return m, nil
}

func (m *model) handleFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.Escape) {
		m.screen = screenHostList
		return m, nil
	}

	formModel, cmd := m.form.Update(msg)
	m.form = formModel.(*huh.Form)

	if m.form.State == huh.StateCompleted {
		return m, m.saveAndGoBack()
	}

	return m, cmd
}

func (m *model) handleDeleteConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.Escape) {
		m.screen = screenHostList
		return m, nil
	}

	switch strings.ToLower(msg.String()) {
	case "y":
		return m, m.confirmDelete()
	case "n":
		m.screen = screenHostList
		return m, nil
	}
	return m, nil
}

func (m *model) handleErrorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.screen = screenHostList
	return m, nil
}
