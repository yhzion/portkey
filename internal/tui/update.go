package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

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

	case updateAvailableMsg:
		m.updateTag = msg.Tag
		m.latestRelease = msg.Rel
		return m, nil

	case updateCheckFailedMsg:
		return m, nil

	case updateDoneMsg:
		if msg.err != nil {
			m.errMsg = fmt.Sprintf("Update failed: %s", msg.err.Error())
			m.screen = screenError
		} else {
			m.errMsg = "Update successful. Please restart portkey."
			m.screen = screenError
		}
		return m, nil

	case errMsg:
		m.errMsg = msg.Error()
		m.screen = screenError
		return m, nil

	case nil:
		return m, nil

	default:
		// Forward huh-internal messages (nextFieldMsg, prevFieldMsg,
		// nextGroupMsg, updateFieldMsg, etc.) to the form when on a
		// form screen. These are produced by form commands and must be
		// routed back for field navigation to work.
		if m.screen == screenAddHost || m.screen == screenEditHost {
			return m.forwardToForm(msg)
		}
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
	case screenUpdateConfirm:
		return m.handleUpdateConfirmKey(msg)
	case screenError:
		return m.handleErrorKey(msg)
	}
	return m, nil
}

func (m *model) handleHostListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// When search is active, handle search input first.
	if m.search.active {
		return m.handleSearchKey(msg)
	}

	switch {
	case key.Matches(msg, m.keys.Search):
		m.activateSearch()
		return m, nil
	case msg.String() == "s":
		m.activateSearch()
		return m, nil
	}

	totalItems := m.totalItems()

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
	case msg.String() == "u":
		if m.latestRelease != nil {
			m.screen = screenUpdateConfirm
			return m, nil
		}
	case msg.String() == "a":
		return m, m.showAddScreen()
	case key.Matches(msg, m.keys.Edit):
		if idx := m.selectedHostIndex(); idx >= 0 {
			return m, m.showEditScreen(idx)
		}
	case key.Matches(msg, m.keys.Delete):
		if idx := m.selectedHostIndex(); idx >= 0 {
			m.showDeleteConfirm(idx)
		}
	case key.Matches(msg, m.keys.Enter), key.Matches(msg, m.keys.Space):
		return m.handleSelect()
	default:
		if len(msg.String()) == 1 && msg.String() >= "1" && msg.String() <= "9" {
			num := int(msg.String()[0] - '0')
			visible := m.visibleHosts()
			if num <= len(visible) {
				return m, m.connectHost(visible[num-1])
			}
		}
	}
	return m, nil
}

func (m *model) handleSelect() (tea.Model, tea.Cmd) {
	if m.selected >= len(m.visibleHosts()) {
		return m, m.showAddScreen()
	}
	if idx := m.selectedHostIndex(); idx >= 0 {
		return m, m.connectHost(idx)
	}
	return m, nil
}

func (m *model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape):
		m.deactivateSearch()
		return m, nil
	case msg.Type == tea.KeyBackspace:
		runes := []rune(m.search.query)
		if len(runes) > 0 {
			m.search.query = string(runes[:len(runes)-1])
			m.updateFilter()
		} else {
			m.deactivateSearch()
		}
		return m, nil
	case key.Matches(msg, m.keys.Up):
		if m.selected > 0 {
			m.selected--
		}
		return m, nil
	case key.Matches(msg, m.keys.Down):
		if m.selected < m.totalItems()-1 {
			m.selected++
		}
		return m, nil
	case key.Matches(msg, m.keys.Enter), key.Matches(msg, m.keys.Space):
		// Select from filtered results.
		result, cmd := m.handleSelect()
		m.deactivateSearch()
		return result, cmd
	case msg.String() == "q":
		// Allow quit from search.
		return m, tea.Quit
	default:
		// Quick-connect numbers (1-9) on filtered results.
		r := msg.String()
		if len(r) == 1 && r >= "1" && r <= "9" {
			visible := m.visibleHosts()
			num := int(r[0] - '0')
			if num <= len(visible) {
				cmd := m.connectHost(visible[num-1])
				m.deactivateSearch()
				return m, cmd
			}
		}
		// Printable character: append to search query.
		if len(r) == 1 && r[0] >= 32 && r[0] < 127 {
			m.search.query += r
			m.updateFilter()
		}
	}
	return m, nil
}

// forwardToForm passes a message to the huh form and handles the result.
// This is used for huh-internal messages that drive field navigation.
func (m *model) forwardToForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.form == nil {
		return m, nil
	}
	formModel, cmd := m.form.Update(msg)
	m.form = formModel.(*huh.Form)
	if m.form.State == huh.StateCompleted {
		return m, m.saveAndGoBack()
	}
	if m.form.State == huh.StateAborted {
		m.screen = screenHostList
		return m, nil
	}
	return m, cmd
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
	if m.form.State == huh.StateAborted {
		m.screen = screenHostList
		return m, nil
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

func (m *model) handleUpdateConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.Escape) {
		m.screen = screenHostList
		return m, nil
	}

	switch strings.ToLower(msg.String()) {
	case "y":
		m.screen = screenHostList
		return m, func() tea.Msg {
			return updateDoneMsg{err: nil}
		}
	case "n":
		m.screen = screenHostList
		return m, nil
	}
	return m, nil
}
