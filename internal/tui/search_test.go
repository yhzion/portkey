package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yhzion/portkey/internal/config"
)

// --- Search activation/deactivation ---

func TestSearch_SlashKey_ActivatesSearch(t *testing.T) {
	m := newTestModel(testHostDev)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !m.search.active {
		t.Error("'/ should activate search")
	}
	if m.search.query != "" {
		t.Errorf("searchQuery = %q, want empty on activation", m.search.query)
	}
}

func TestSearch_SKey_ActivatesSearch(t *testing.T) {
	m := newTestModel(testHostDev)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if !m.search.active {
		t.Error("'s' should activate search")
	}
}

func TestSearch_Esc_DeactivatesSearch(t *testing.T) {
	m := newTestModel(testHostDev)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.search.active {
		t.Error("esc should deactivate search")
	}
	if m.search.query != "" {
		t.Error("esc should clear search query")
	}
}

func TestSearch_BackspaceOnEmpty_Deactivates(t *testing.T) {
	m := newTestModel(testHostDev)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.search.active {
		t.Error("backspace on empty query should deactivate search")
	}
}

// --- Search typing ---

func TestSearch_TypeChars_AppendsToQuery(t *testing.T) {
	m := newTestModel(testHostDev)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	if m.search.query != "pro" {
		t.Errorf("searchQuery = %q, want %q", m.search.query, "pro")
	}
}

func TestSearch_Backspace_RemovesLastChar(t *testing.T) {
	m := newTestModel(testHostDev)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.search.query != "p" {
		t.Errorf("searchQuery = %q, want %q", m.search.query, "p")
	}
}

// --- Search filtering ---

func TestSearch_FiltersResults(t *testing.T) {
	hosts := []config.Host{
		{Name: "production-api", Username: "u", Host: "h1", Port: 22},
		{Name: "staging", Username: "u", Host: "h2", Port: 22},
		{Name: "prod", Username: "u", Host: "h3", Port: 22},
	}
	m := newTestModel(hosts...)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})

	if len(m.search.filtered) != 2 {
		t.Errorf("len(filtered) = %d, want 2 (production-api, prod)", len(m.search.filtered))
	}
}

func TestSearch_NoMatches(t *testing.T) {
	m := newTestModel(testHostDev)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})

	if len(m.search.filtered) != 0 {
		t.Errorf("len(filtered) = %d, want 0 for no match", len(m.search.filtered))
	}
}

// --- Search navigation ---

func TestSearch_NavigationOnFilteredResults(t *testing.T) {
	hosts := []config.Host{
		{Name: "alpha", Username: "u", Host: "h1", Port: 22},
		{Name: "beta", Username: "u", Host: "h2", Port: 22},
		{Name: "gamma", Username: "u", Host: "h3", Port: 22},
	}
	m := newTestModel(hosts...)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	// Type 'a' — all three match
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	if m.selected != 0 {
		t.Errorf("selected = %d, want 0 after activating search", m.selected)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.selected != 1 {
		t.Errorf("selected = %d, want 1 after down in filtered", m.selected)
	}
}

// --- Search + connect ---

func TestSearch_EnterSelectsFilteredHost(t *testing.T) {
	hosts := []config.Host{
		{Name: "alpha", Username: "u", Host: "h1", Port: 22},
		{Name: "beta", Username: "u", Host: "h2", Port: 22},
	}
	m := newTestModel(hosts...)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}}) // matches "beta"
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("enter on filtered host should trigger connect")
	}
	if m.search.active {
		t.Error("search should deactivate after selecting")
	}
}

// --- Search quick-connect on filtered results ---

func TestSearch_QuickConnect_FilteredResults(t *testing.T) {
	hosts := []config.Host{
		{Name: "production-api", Username: "u", Host: "h1", Port: 22},
		{Name: "staging", Username: "u", Host: "h2", Port: 22},
		{Name: "prod", Username: "u", Host: "h3", Port: 22},
	}
	m := newTestModel(hosts...)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}) // matches production-api, prod

	// Quick-connect 1 should connect to first filtered result
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	if cmd == nil {
		t.Error("quick-connect 1 on filtered should trigger connect")
	}
}

// --- Search doesn't interfere with normal mode ---

func TestSearch_NotActive_NormalKeysWork(t *testing.T) {
	m := newTestModel(testHostDev)
	// Without activating search, 'e' should edit, 'd' should delete etc.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if m.screen != screenEditHost {
		t.Error("e should still open edit when search not active")
	}
}

func TestSearch_Deactivated_RestoresFullList(t *testing.T) {
	hosts := []config.Host{
		{Name: "alpha", Username: "u", Host: "h1", Port: 22},
		{Name: "beta", Username: "u", Host: "h2", Port: 22},
	}
	m := newTestModel(hosts...)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if m.search.active {
		t.Error("search should be deactivated")
	}
	visible := m.visibleHosts()
	if len(visible) != 2 {
		t.Errorf("visible hosts = %d, want 2 after deactivating search", len(visible))
	}
}

// --- View rendering for search ---

func TestView_SearchBarShown(t *testing.T) {
	m := newTestModel(testHostDev)
	m.activateSearch()
	m.search.query = "pro"
	view := m.View()

	if !strings.Contains(view, "/>") {
		t.Error("search bar should show '/>' prompt")
	}
	if !strings.Contains(view, "pro") {
		t.Error("search bar should show query text")
	}
}

func TestView_SearchNoMatches(t *testing.T) {
	m := newTestModel(testHostDev)
	m.activateSearch()
	m.search.query = "zzz"
	m.updateFilter()
	view := m.View()

	if !strings.Contains(view, "No matches") {
		t.Error("should show 'No matches' when no results")
	}
}

func TestView_SearchHelpBar(t *testing.T) {
	m := newTestModel(testHostDev)
	m.activateSearch()
	view := m.View()

	if !strings.Contains(view, "type to filter") {
		t.Error("search mode help should show filter hint")
	}
	if !strings.Contains(view, "esc clear") {
		t.Error("search mode help should show esc hint")
	}
}

func TestView_SearchShowsFilteredHosts(t *testing.T) {
	hosts := []config.Host{
		{Name: "production-api", Username: "u", Host: "h1", Port: 22},
		{Name: "staging", Username: "u", Host: "h2", Port: 22},
	}
	m := newTestModel(hosts...)
	m.activateSearch()
	m.search.query = "pro"
	m.updateFilter()
	view := m.View()

	if !strings.Contains(view, "production-api") {
		t.Error("filtered view should show matching host")
	}
	if strings.Contains(view, "staging") {
		t.Error("filtered view should not show non-matching host")
	}
}

func TestView_NormalMode_HelpShowsSearchKey(t *testing.T) {
	m := newTestModel(testHostDev)
	view := m.View()

	if !strings.Contains(view, "/ search") {
		t.Error("normal mode help should show '/ search'")
	}
}

// --- Search quits ---

func TestSearch_QKey_Quits(t *testing.T) {
	m := newTestModel(testHostDev)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Error("q during search should quit")
	}
}

// --- Number out of range appends to query ---

func TestSearch_NumberOutOfRange_AppendsToQuery(t *testing.T) {
	m := newTestModel(testHostDev) // only 1 host
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	// 9 is out of range (only 1 host), should append to query
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'9'}})
	if m.search.query != "9" {
		t.Errorf("searchQuery = %q, want %q (number out of range should append)", m.search.query, "9")
	}
}
