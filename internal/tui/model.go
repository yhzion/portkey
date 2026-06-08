package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/yhzion/portkey/internal/config"
	"github.com/yhzion/portkey/internal/updater"
)

type screen int

const (
	screenHostList screen = iota
	screenAddHost
	screenEditHost
	screenDeleteConfirm
	screenUpdateConfirm
	screenError
)

type hostForm struct {
	Name     string
	Username string
	Host     string
	Port     string
}

func (f *hostForm) toHost() config.Host {
	port := 22
	if f.Port != "" {
		fmt.Sscanf(f.Port, "%d", &port)
	}
	name := f.Name
	if name == "" {
		name = f.Username
	}
	return config.Host{
		Name:     name,
		Username: f.Username,
		Host:     f.Host,
		Port:     port,
	}
}

type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

type updateAvailableMsg struct {
	Tag string
	Rel *updater.Release
}

type updateCheckFailedMsg struct{}

type updateDoneMsg struct {
	err error
}

// SearchModel encapsulates the host-list search/filter state.
type SearchModel struct {
	Active   bool
	Query    string
	Filtered []MatchResult // fuzzy match results for current query
}

// Activate enters search mode and seeds the filter with all hosts.
func (s *SearchModel) Activate(hosts []config.Host) {
	s.Active = true
	s.Query = ""
	s.Filtered = FuzzyMatch(hosts, "")
}

// Deactivate exits search mode and clears the filter.
func (s *SearchModel) Deactivate() {
	s.Active = false
	s.Query = ""
	s.Filtered = nil
}

// UpdateFilter re-runs fuzzy match for the current query.
func (s *SearchModel) UpdateFilter(hosts []config.Host) {
	s.Filtered = FuzzyMatch(hosts, s.Query)
}

type model struct {
	screen        screen
	config        *config.Config
	selected      int
	form          *huh.Form
	hostForm      *hostForm
	editIndex     int
	errMsg        string
	keys          keyMap
	width         int
	height        int
	updateTag     string
	latestRelease *updater.Release
	Version       string
	Updater       *updater.Client

	// store persists config. Provided via dependency injection.
	store config.Store

	// Search/filter state
	search SearchModel

	// Last-connected tracking
	connectIndex int  // index of host being connected (-1 = none)
	connected    bool // true after connectHost is called
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
	Search key.Binding
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
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
	}
}

func InitialModel(cfg *config.Config, version string, upd *updater.Client, store config.Store) tea.Model {
	m := &model{
		screen:   screenHostList,
		config:   cfg,
		selected: 0,
		keys:     newKeyMap(),
		Version:  version,
		Updater:  upd,
		store:    store,
	}
	return m
}

func (m *model) Init() tea.Cmd {
	config.SortHosts(m.config.Hosts)
	if m.Updater != nil && m.Version != "dev" {
		return m.checkUpdate()
	}
	return nil
}

func (m *model) checkUpdate() tea.Cmd {
	return func() tea.Msg {
		rel, err := m.Updater.CheckLatest()
		if err != nil {
			return updateCheckFailedMsg{}
		}
		if updater.IsNewer(m.Version, rel.Tag) {
			return updateAvailableMsg{Tag: rel.Tag, Rel: rel}
		}
		return nil
	}
}

func buildHostForm(hf *hostForm) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Name").
				Value(&hf.Name).
				Validate(func(s string) error {
					if err := config.ValidateName(s); err != nil {
						return err
					}
					return nil
				}),
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
		Name:     h.Name,
		Username: h.Username,
		Host:     h.Host,
		Port:     portStr,
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
		if err := m.store.Save(m.config); err != nil {
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
		if err := m.store.Save(m.config); err != nil {
			return errMsg{err}
		}
		return nil
	}
}

func (m *model) connectHost(index int) tea.Cmd {
	m.connectIndex = index
	m.connected = true
	return tea.Quit
}

// visibleHosts returns the host indices to display.
// When search is active, returns filtered results; otherwise all hosts.
func (m *model) visibleHosts() []int {
	if m.search.Active {
		indices := make([]int, len(m.search.Filtered))
		for i, r := range m.search.Filtered {
			indices[i] = r.HostIndex
		}
		return indices
	}
	indices := make([]int, len(m.config.Hosts))
	for i := range m.config.Hosts {
		indices[i] = i
	}
	return indices
}

// totalItems returns the number of selectable items (hosts + "Add" item).
func (m *model) totalItems() int {
	return len(m.visibleHosts()) + 1
}

// selectedHostIndex returns the config.Hosts index of the currently selected item.
// Returns -1 if the selection is on the "Add" item.
func (m *model) selectedHostIndex() int {
	visible := m.visibleHosts()
	if m.selected >= len(visible) {
		return -1 // "Add" item or out of range
	}
	return visible[m.selected]
}

// updateFilter runs fuzzy match and resets selection.
func (m *model) updateFilter() {
	m.search.UpdateFilter(m.config.Hosts)
	if m.selected >= len(m.search.Filtered)+1 {
		m.selected = max(0, len(m.search.Filtered))
	}
}

// activateSearch enters search mode.
func (m *model) activateSearch() {
	m.search.Activate(m.config.Hosts)
	m.selected = 0
}

// deactivateSearch exits search mode and restores full list.
func (m *model) deactivateSearch() {
	m.search.Deactivate()
	m.selected = 0
}

// SelectedHost extracts the selected host from a tea.Model after the TUI
// exits. Returns the host and true if a host was selected, or false if
// the user quit without selecting.
func SelectedHost(m tea.Model) (*config.Host, bool) {
	tui, ok := m.(*model)
	if !ok {
		return nil, false
	}
	if tui.connectIndex < 0 || tui.connectIndex >= len(tui.config.Hosts) {
		return nil, false
	}
	// connectIndex 0 is ambiguous: could be unset (Go zero) or host 0.
	// Use a sentinel: connectIndex is only valid after connectHost sets it.
	// We track this via the connected field.
	if !tui.connected {
		return nil, false
	}
	host := tui.config.Hosts[tui.connectIndex]
	return &host, true
}
