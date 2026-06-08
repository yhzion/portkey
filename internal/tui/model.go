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
	screenUpdateConfirm
	screenError
	screenNotification
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

// UpdateChecker reports the latest available release. Defined in the consumer
// package so the model can be tested with a fake instead of a live HTTP client.
// *updater.Client satisfies it.
type UpdateChecker interface {
	CheckLatest() (*updater.Release, error)
}

type updateDoneMsg struct {
	err error
}

// formModel encapsulates the add/edit host form state.
type formModel struct {
	form     *huh.Form
	hostForm *hostForm
}

// showAdd builds an empty add-host form and returns its init command.
func (f *formModel) showAdd() tea.Cmd {
	f.hostForm = &hostForm{Port: "22"}
	f.form = buildHostForm(f.hostForm)
	return f.form.Init()
}

// showEdit builds an edit form pre-filled from the given host.
func (f *formModel) showEdit(h config.Host) tea.Cmd {
	portStr := ""
	if h.Port != 0 {
		portStr = fmt.Sprintf("%d", h.Port)
	}
	if portStr == "" {
		portStr = "22"
	}
	f.hostForm = &hostForm{
		Name:     h.Name,
		Username: h.Username,
		Host:     h.Host,
		Port:     portStr,
	}
	f.form = buildHostForm(f.hostForm)
	return f.form.Init()
}

// resize adjusts the form width to fit the terminal.
func (f *formModel) resize(width int) {
	if f.form != nil {
		f.form = f.form.WithWidth(min(width-4, 80))
	}
}

// update forwards a message to the huh form and returns its resulting state.
// Returns StateNormal with no command when no form is active.
func (f *formModel) update(msg tea.Msg) (huh.FormState, tea.Cmd) {
	if f.form == nil {
		return huh.StateNormal, nil
	}
	updated, cmd := f.form.Update(msg)
	f.form = updated.(*huh.Form)
	return f.form.State, cmd
}

// searchModel encapsulates the host-list search/filter state.
type searchModel struct {
	active   bool
	query    string
	filtered []matchResult // fuzzy match results for current query
}

// activate enters search mode and seeds the filter with all hosts.
func (s *searchModel) activate(hosts []config.Host) {
	s.active = true
	s.query = ""
	s.filtered = fuzzyMatch(hosts, "")
}

// deactivate exits search mode and clears the filter.
func (s *searchModel) deactivate() {
	s.active = false
	s.query = ""
	s.filtered = nil
}

// updateFilter re-runs fuzzy match for the current query.
func (s *searchModel) updateFilter(hosts []config.Host) {
	s.filtered = fuzzyMatch(hosts, s.query)
}

// indices returns the host indices in display order for the current filter.
// Returns nil when search is inactive.
func (s *searchModel) indices() []int {
	if !s.active {
		return nil
	}
	indices := make([]int, len(s.filtered))
	for i, r := range s.filtered {
		indices[i] = r.hostIndex
	}
	return indices
}

// matchMap returns a map from host index to its match result, used for
// highlight positions in the rendered host list. Returns nil when search
// is inactive. Map values point into s.filtered and are valid as long as
// the slice is not reallocated.
func (s *searchModel) matchMap() map[int]*matchResult {
	if !s.active || len(s.filtered) == 0 {
		return nil
	}
	m := make(map[int]*matchResult, len(s.filtered))
	for i := range s.filtered {
		m[s.filtered[i].hostIndex] = &s.filtered[i]
	}
	return m
}

type model struct {
	screen    screen
	config    *config.Config
	selected  int
	editIndex int // target row for edit/delete modal ops (shared by form + delete)
	errMsg    string
	keys      keyMap
	width     int
	height    int

	// Update state
	updateModel updateModel

	// store persists config. Provided via dependency injection.
	store config.Store

	// Add/edit form state
	formModel formModel

	// Search/filter state
	search searchModel

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

func InitialModel(cfg *config.Config, version string, upd UpdateChecker, store config.Store) tea.Model {
	if client, ok := upd.(*updater.Client); ok && client == nil {
		upd = nil
	}
	m := &model{
		screen:   screenHostList,
		config:   cfg,
		selected: 0,
		keys:     newKeyMap(),
		store:    store,
		updateModel: updateModel{
			version: version,
			checker: upd,
		},
	}
	return m
}

func (m *model) Init() tea.Cmd {
	config.SortHosts(m.config.Hosts)
	if m.updateModel.checker != nil && m.updateModel.version != "dev" {
		return m.updateModel.checkUpdate()
	}
	return nil
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
	return m.formModel.showAdd()
}

func (m *model) showEditScreen(index int) tea.Cmd {
	m.screen = screenEditHost
	m.editIndex = index
	return m.formModel.showEdit(m.config.Hosts[index])
}

func (m *model) showDeleteConfirm(index int) {
	m.screen = screenDeleteConfirm
	m.editIndex = index
}

func (m *model) saveAndGoBack() tea.Cmd {
	host := m.formModel.hostForm.toHost()

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
	if indices := m.search.indices(); indices != nil {
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
	m.search.updateFilter(m.config.Hosts)
	if m.selected >= len(m.search.filtered)+1 {
		m.selected = max(0, len(m.search.filtered))
	}
}

// activateSearch enters search mode.
func (m *model) activateSearch() {
	m.search.activate(m.config.Hosts)
	m.selected = 0
}

// deactivateSearch exits search mode and restores full list.
func (m *model) deactivateSearch() {
	m.search.deactivate()
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
