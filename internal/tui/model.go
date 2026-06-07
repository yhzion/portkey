package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/yhzion/portkey/internal/config"
)

type screen int

const (
	screenHostList screen = iota
	screenAddHost
	screenEditHost
	screenDeleteConfirm
	screenError
)

type hostForm struct {
	DisplayName string
	Username    string
	Host        string
	Port        string
}

func (f *hostForm) toHost() config.Host {
	port := 22
	if f.Port != "" {
		fmt.Sscanf(f.Port, "%d", &port)
	}
	displayName := f.DisplayName
	if displayName == "" {
		displayName = f.Username
	}
	return config.Host{
		DisplayName: displayName,
		Username:    f.Username,
		Host:        f.Host,
		Port:        port,
	}
}

type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

type sshDoneMsg struct{ err error }

type model struct {
	screen    screen
	config    *config.Config
	selected  int
	form      *huh.Form
	hostForm  *hostForm
	editIndex int
	errMsg    string
	keys      keyMap
	width     int
	height    int
}

type keyMap struct {
	Up     key.Binding
	Down   key.Binding
	Enter  key.Binding
	Space  key.Binding
	Quit   key.Binding
	Add    key.Binding
	Edit   key.Binding
	Delete key.Binding
	Escape key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "move down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Space: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "select"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Add: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add host"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
	}
}

func InitialModel(cfg *config.Config) tea.Model {
	m := &model{
		screen:   screenHostList,
		config:   cfg,
		selected: 0,
		keys:     newKeyMap(),
	}
	return m
}

func (m *model) Init() tea.Cmd {
	return nil
}

func buildHostForm(hf *hostForm) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Display Name").
				Value(&hf.DisplayName).
				Validate(func(s string) error { return nil }),
			huh.NewInput().
				Title("Username").
				Value(&hf.Username).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("username is required")
					}
					return nil
				}),
			huh.NewInput().
				Title("Host").
				Value(&hf.Host).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("host is required")
					}
					return nil
				}),
			huh.NewInput().
				Title("Port").
				Placeholder("22").
				Value(&hf.Port).
				Validate(func(s string) error {
					if s == "" {
						return nil
					}
					var port int
					if _, err := fmt.Sscanf(s, "%d", &port); err != nil {
						return fmt.Errorf("port must be a number")
					}
					if port < 1 || port > 65535 {
						return fmt.Errorf("port must be between 1 and 65535")
					}
					return nil
				}),
		),
	).WithShowHelp(false)
}

func (m *model) showAddScreen() tea.Cmd {
	m.screen = screenAddHost
	m.hostForm = &hostForm{Port: "22"}
	m.form = buildHostForm(m.hostForm)
	return m.form.Init()
}

func (m *model) showEditScreen(index int) tea.Cmd {
	m.screen = screenEditHost
	m.editIndex = index
	h := m.config.Hosts[index]
	portStr := ""
	if h.Port != 0 {
		portStr = fmt.Sprintf("%d", h.Port)
	}
	if portStr == "" {
		portStr = "22"
	}
	m.hostForm = &hostForm{
		DisplayName: h.DisplayName,
		Username:    h.Username,
		Host:        h.Host,
		Port:        portStr,
	}
	m.form = buildHostForm(m.hostForm)
	return m.form.Init()
}

func (m *model) showDeleteConfirm(index int) {
	m.screen = screenDeleteConfirm
	m.editIndex = index
}

func (m *model) saveAndGoBack() tea.Cmd {
	host := m.hostForm.toHost()

	if m.screen == screenAddHost {
		m.config.AddHost(host)
	} else if m.screen == screenEditHost {
		m.config.UpdateHost(m.editIndex, host)
	}

	m.screen = screenHostList
	m.selected = 0
	return func() tea.Msg {
		if err := config.Save(m.config); err != nil {
			return errMsg{err}
		}
		return nil
	}
}

func (m *model) confirmDelete() tea.Cmd {
	m.config.RemoveHost(m.editIndex)
	m.screen = screenHostList
	m.selected = 0
	if m.selected >= len(m.config.Hosts) {
		m.selected = len(m.config.Hosts)
	}
	return func() tea.Msg {
		if err := config.Save(m.config); err != nil {
			return errMsg{err}
		}
		return nil
	}
}

func (m *model) connectHost(index int) tea.Cmd {
	return func() tea.Msg {
		err := sshRun(m.config.Hosts[index])
		return sshDoneMsg{err: err}
	}
}
