package tui

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yhzion/portkey/internal/config"
	"github.com/yhzion/portkey/internal/updater"
)

// ---------------------------------------------------------------------------
// Test helpers: simulating the Bubble Tea runtime message pump
// ---------------------------------------------------------------------------

// pumpCmds executes a tea.Cmd and feeds the resulting message back to the
// model, recursively, simulating the Bubble Tea runtime. A depth limit
// prevents infinite loops from self-scheduling commands (cursor blink).
func pumpCmds(t *testing.T, m *model, cmd tea.Cmd) {
	t.Helper()
	pumpCmdsN(t, m, cmd, 30)
}

func pumpCmdsN(t *testing.T, m *model, cmd tea.Cmd, limit int) {
	t.Helper()
	if limit <= 0 || cmd == nil {
		return
	}

	// Cursor blink commands block for the blink duration (~500ms).
	// Use a goroutine + timeout so tests stay fast.
	var msg tea.Msg
	done := make(chan struct{})
	go func() {
		msg = cmd()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(50 * time.Millisecond):
		// Command blocked (cursor blink timer) — skip it.
		return
	}

	if msg == nil {
		return
	}

	// tea.batchMsg and tea.sequenceMsg are both []tea.Cmd under the hood.
	rv := reflect.ValueOf(msg)
	if rv.Kind() == reflect.Slice && rv.Len() > 0 && rv.Index(0).Kind() == reflect.Func {
		for i := 0; i < rv.Len(); i++ {
			if fn, ok := rv.Index(i).Interface().(tea.Cmd); ok {
				pumpCmdsN(t, m, fn, limit-1)
			}
		}
		return
	}

	_, nextCmd := m.Update(msg)
	pumpCmdsN(t, m, nextCmd, limit-1)
}

// typeRunes sends one KeyMsg per rune. Ignores returned commands (rendering only).
func typeRunes(m *model, runes []rune) {
	for _, r := range runes {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
}

func typeString(m *model, s string) {
	typeRunes(m, []rune(s))
}

// Keyboard interaction helpers that pump the command loop.
func pressEnter(t *testing.T, m *model) {
	t.Helper()
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pumpCmds(t, m, cmd)
}

func pressTab(t *testing.T, m *model) {
	t.Helper()
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	pumpCmds(t, m, cmd)
}

func pressShiftTab(t *testing.T, m *model) {
	t.Helper()
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	pumpCmds(t, m, cmd)
}

func pressKey(m *model, r rune) {
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
}

func pressEsc(m *model) {
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
}

func pressUp(m *model) {
	m.Update(tea.KeyMsg{Type: tea.KeyUp})
}

func pressDown(m *model) {
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
}

// showAddForm opens the add-host screen and processes init commands.
func showAddForm(t *testing.T, m *model) {
	t.Helper()
	cmd := m.showAddScreen()
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	pumpCmds(t, m, cmd)
}

// showEditForm opens the edit-host screen for the host at the given index.
func showEditForm(t *testing.T, m *model, index int) {
	t.Helper()
	cmd := m.showEditScreen(index)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	pumpCmds(t, m, cmd)
}

// fillAddForm types values into all four fields, advancing with Enter.
// Name → Username → Host → Port. Clears the default port before typing.
func fillAddForm(t *testing.T, m *model, name, username, host, port string) {
	t.Helper()
	typeString(m, name)
	pressEnter(t, m)
	typeString(m, username)
	pressEnter(t, m)
	typeString(m, host)
	pressEnter(t, m)
	// Clear the default port value ("22") before typing the new one.
	portLen := len(m.hostForm.Port)
	for i := 0; i < portLen; i++ {
		m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	}
	typeString(m, port)
}

// submitAddForm presses Enter on the last field to submit the form.
func submitAddForm(t *testing.T, m *model) {
	t.Helper()
	pressEnter(t, m)
}

// viewContains checks that the model's View() output contains all substrings.
func viewContains(t *testing.T, m *model, substrs ...string) {
	t.Helper()
	view := m.View()
	for _, s := range substrs {
		if !strings.Contains(view, s) {
			t.Errorf("view does not contain %q.\nview:\n%s", s, view)
		}
	}
}

func viewNotContains(t *testing.T, m *model, substrs ...string) {
	t.Helper()
	view := m.View()
	for _, s := range substrs {
		if strings.Contains(view, s) {
			t.Errorf("view should not contain %q.\nview:\n%s", s, view)
		}
	}
}

// ---------------------------------------------------------------------------
// E2E: Add Host — full flow and variations
// ---------------------------------------------------------------------------

func TestE2E_AddHost_FullFlow(t *testing.T) {
	m := newTestModel()
	showAddForm(t, m)

	fillAddForm(t, m, "myserver", "alice", "10.0.0.1", "2222")
	submitAddForm(t, m)

	if m.screen != screenHostList {
		t.Fatalf("screen = %d, want screenHostList after submit", m.screen)
	}
	if len(m.config.Hosts) != 1 {
		t.Fatalf("len(Hosts) = %d, want 1", len(m.config.Hosts))
	}
	h := m.config.Hosts[0]
	if h.Name != "myserver" {
		t.Errorf("Name = %q, want %q", h.Name, "myserver")
	}
	if h.Username != "alice" {
		t.Errorf("Username = %q, want %q", h.Username, "alice")
	}
	if h.Host != "10.0.0.1" {
		t.Errorf("Host = %q, want %q", h.Host, "10.0.0.1")
	}
	if h.Port != 2222 {
		t.Errorf("Port = %d, want 2222", h.Port)
	}
}

func TestE2E_AddHost_CancelEscape(t *testing.T) {
	m := newTestModel()
	showAddForm(t, m)

	typeString(m, "myserver")
	pressEsc(m)

	if m.screen != screenHostList {
		t.Errorf("screen = %d, want screenHostList after esc", m.screen)
	}
	if len(m.config.Hosts) != 0 {
		t.Errorf("len(Hosts) = %d, want 0 after cancel", len(m.config.Hosts))
	}
}

func TestE2E_AddHost_DefaultPort22(t *testing.T) {
	m := newTestModel()
	showAddForm(t, m)

	// Port defaults to "22" in the form; just press Enter on it without typing.
	typeString(m, "srv")
	pressEnter(t, m)
	typeString(m, "bob")
	pressEnter(t, m)
	typeString(m, "example.com")
	pressEnter(t, m)
	// Port field still has "22" — submit
	submitAddForm(t, m)

	if len(m.config.Hosts) != 1 {
		t.Fatalf("len(Hosts) = %d, want 1", len(m.config.Hosts))
	}
	if m.config.Hosts[0].Port != 22 {
		t.Errorf("Port = %d, want 22 (default)", m.config.Hosts[0].Port)
	}
}

func TestE2E_AddHost_ThenAddAnother(t *testing.T) {
	m := newTestModel()

	// Add first host.
	showAddForm(t, m)
	fillAddForm(t, m, "alpha", "user", "host1", "22")
	submitAddForm(t, m)
	if len(m.config.Hosts) != 1 {
		t.Fatalf("len(Hosts) = %d after first add, want 1", len(m.config.Hosts))
	}

	// Add second host via 'a' key from host list.
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	pumpCmds(t, m, cmd)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	fillAddForm(t, m, "beta", "user", "host2", "22")
	submitAddForm(t, m)

	if len(m.config.Hosts) != 2 {
		t.Fatalf("len(Hosts) = %d after second add, want 2", len(m.config.Hosts))
	}
}

// ---------------------------------------------------------------------------
// E2E: Form field navigation (the bug fix)
// ---------------------------------------------------------------------------

func TestE2E_FormNav_EnterAdvancesToNextField(t *testing.T) {
	m := newTestModel()
	showAddForm(t, m)

	// Type into Name field.
	typeString(m, "myserver")

	// Press Enter — should move to Username field.
	pressEnter(t, m)

	// Type into what should now be the Username field.
	typeString(m, "alice")

	// If navigation worked, hostForm.Name="myserver" and hostForm.Username="alice".
	// If broken (everything stays on Name), Name="myserveralice" and Username="".
	if m.hostForm.Name != "myserver" {
		t.Errorf("Name = %q, want %q", m.hostForm.Name, "myserver")
	}
	if m.hostForm.Username != "alice" {
		t.Errorf("Username = %q, want %q — Enter did not advance field", m.hostForm.Username, "alice")
	}
}

func TestE2E_FormNav_TabAdvancesToNextField(t *testing.T) {
	m := newTestModel()
	showAddForm(t, m)

	typeString(m, "myserver")
	pressTab(t, m)
	typeString(m, "alice")

	if m.hostForm.Name != "myserver" {
		t.Errorf("Name = %q, want %q", m.hostForm.Name, "myserver")
	}
	if m.hostForm.Username != "alice" {
		t.Errorf("Username = %q, want %q — Tab did not advance field", m.hostForm.Username, "alice")
	}
}

func TestE2E_FormNav_ShiftTabGoesBack(t *testing.T) {
	m := newTestModel()
	showAddForm(t, m)

	// Fill Name, advance to Username.
	typeString(m, "srv")
	pressEnter(t, m)

	// Go back to Name.
	pressShiftTab(t, m)

	// Type more — should append to Name, not Username.
	typeString(m, "2")
	if m.hostForm.Name != "srv2" {
		t.Errorf("Name = %q, want %q after Shift+Tab back", m.hostForm.Name, "srv2")
	}
	if m.hostForm.Username != "" {
		t.Errorf("Username = %q, want empty (should still be on Name)", m.hostForm.Username)
	}
}

func TestE2E_FormNav_WalkAllFieldsForward(t *testing.T) {
	m := newTestModel()
	showAddForm(t, m)

	typeString(m, "srv")
	pressEnter(t, m)
	typeString(m, "alice")
	pressEnter(t, m)
	typeString(m, "10.0.0.1")
	pressEnter(t, m)
	// Clear default "22" and type custom port.
	portLen := len(m.hostForm.Port)
	for i := 0; i < portLen; i++ {
		m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	}
	typeString(m, "2222")

	if m.hostForm.Name != "srv" {
		t.Errorf("Name = %q, want %q", m.hostForm.Name, "srv")
	}
	if m.hostForm.Username != "alice" {
		t.Errorf("Username = %q, want %q", m.hostForm.Username, "alice")
	}
	if m.hostForm.Host != "10.0.0.1" {
		t.Errorf("Host = %q, want %q", m.hostForm.Host, "10.0.0.1")
	}
	if m.hostForm.Port != "2222" {
		t.Errorf("Port = %q, want %q", m.hostForm.Port, "2222")
	}
}

func TestE2E_FormNav_EnterOnLastFieldSubmits(t *testing.T) {
	m := newTestModel()
	showAddForm(t, m)

	fillAddForm(t, m, "srv", "alice", "10.0.0.1", "2222")
	submitAddForm(t, m)

	if m.screen != screenHostList {
		t.Errorf("screen = %d, want screenHostList after form submit", m.screen)
	}
	if len(m.config.Hosts) != 1 {
		t.Fatalf("len(Hosts) = %d, want 1 after submit", len(m.config.Hosts))
	}
}

// ---------------------------------------------------------------------------
// E2E: Validation — form blocks on invalid input
// ---------------------------------------------------------------------------

func TestE2E_Validation_InvalidNameBlocksAdvance(t *testing.T) {
	m := newTestModel()
	showAddForm(t, m)

	// Type a name with invalid characters (uppercase).
	typeString(m, "MyServer")
	pressEnter(t, m)

	// Name validation should fail, so we're still on the Name field.
	// Type more to verify we're still here.
	typeString(m, "2")
	if m.hostForm.Name != "MyServer2" {
		t.Errorf("Name = %q, want %q — field did not block on invalid name", m.hostForm.Name, "MyServer2")
	}
	if m.hostForm.Username != "" {
		t.Errorf("Username = %q, want empty — validation should have blocked advance", m.hostForm.Username)
	}
}

func TestE2E_Validation_EmptyNameBlocksAdvance(t *testing.T) {
	m := newTestModel()
	showAddForm(t, m)

	// Press Enter on empty Name.
	pressEnter(t, m)

	// Should still be on Name (validation: name is required).
	typeString(m, "srv")
	if m.hostForm.Name != "srv" {
		t.Errorf("Name = %q, want %q — empty name should block advance", m.hostForm.Name, "srv")
	}
}

func TestE2E_Validation_EmptyUsernameBlocksAdvance(t *testing.T) {
	m := newTestModel()
	showAddForm(t, m)

	typeString(m, "srv")
	pressEnter(t, m) // advance to Username

	// Press Enter on empty Username.
	pressEnter(t, m)

	// Should still be on Username.
	typeString(m, "alice")
	if m.hostForm.Username != "alice" {
		t.Errorf("Username = %q, want %q — empty username should block advance", m.hostForm.Username, "alice")
	}
}

func TestE2E_Validation_EmptyHostBlocksAdvance(t *testing.T) {
	m := newTestModel()
	showAddForm(t, m)

	typeString(m, "srv")
	pressEnter(t, m)
	typeString(m, "alice")
	pressEnter(t, m) // advance to Host

	// Press Enter on empty Host.
	pressEnter(t, m)

	// Should still be on Host.
	typeString(m, "10.0.0.1")
	if m.hostForm.Host != "10.0.0.1" {
		t.Errorf("Host = %q, want %q — empty host should block advance", m.hostForm.Host, "10.0.0.1")
	}
}

func TestE2E_Validation_InvalidPortBlocksAdvance(t *testing.T) {
	m := newTestModel()
	showAddForm(t, m)

	fillAddForm(t, m, "srv", "alice", "10.0.0.1", "")
	// Clear the default port and type invalid value.
	// Port field has "22" by default — type over it.
	// Actually the port field starts as "22", so we need to clear it first.
	// For simplicity, just type an invalid port value after clearing.
	// The form starts with Port="22", let's clear and type "abc".
	for i := 0; i < 2; i++ {
		m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	}
	typeString(m, "abc")
	submitAddForm(t, m)

	// Should NOT have submitted — port "abc" is invalid.
	if m.screen == screenHostList && len(m.config.Hosts) > 0 {
		t.Error("form should not submit with invalid port 'abc'")
	}
}

func TestE2E_Validation_PortOutOfRange(t *testing.T) {
	m := newTestModel()
	showAddForm(t, m)

	fillAddForm(t, m, "srv", "alice", "10.0.0.1", "")
	// Clear default "22" and type "99999"
	for i := 0; i < 2; i++ {
		m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	}
	typeString(m, "99999")
	submitAddForm(t, m)

	if m.screen == screenHostList && len(m.config.Hosts) > 0 {
		t.Error("form should not submit with port 99999")
	}
}

func TestE2E_Validation_ValidBoundaryPorts(t *testing.T) {
	tests := []struct {
		name string
		port string
		want int
	}{
		{"port 1", "1", 1},
		{"port 65535", "65535", 65535},
		{"port 22", "22", 22},
		{"port 8080", "8080", 8080},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModel()
			showAddForm(t, m)

			fillAddForm(t, m, "srv", "alice", "10.0.0.1", "")
			// Clear default port.
			for i := 0; i < 2; i++ {
				m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
			}
			typeString(m, tc.port)
			submitAddForm(t, m)

			if m.screen != screenHostList {
				t.Fatalf("screen = %d, want screenHostList", m.screen)
			}
			if len(m.config.Hosts) != 1 {
				t.Fatalf("len(Hosts) = %d, want 1", len(m.config.Hosts))
			}
			if m.config.Hosts[0].Port != tc.want {
				t.Errorf("Port = %d, want %d", m.config.Hosts[0].Port, tc.want)
			}
		})
	}
}

func TestE2E_Validation_NameWithAllowedChars(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"lowercase", "myserver"},
		{"with hyphen", "my-server"},
		{"with underscore", "my_server"},
		{"with digits", "srv01"},
		{"all allowed", "my-server_v2"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModel()
			showAddForm(t, m)

			fillAddForm(t, m, tc.input, "alice", "10.0.0.1", "22")
			submitAddForm(t, m)

			if m.screen != screenHostList {
				t.Fatalf("screen = %d, want screenHostList for name %q", m.screen, tc.input)
			}
			if len(m.config.Hosts) != 1 {
				t.Fatalf("len(Hosts) = %d, want 1 for name %q", len(m.config.Hosts), tc.input)
			}
			if m.config.Hosts[0].Name != tc.input {
				t.Errorf("Name = %q, want %q", m.config.Hosts[0].Name, tc.input)
			}
		})
	}
}

func TestE2E_Validation_NameWithDisallowedChars(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"uppercase", "MyServer"},
		{"spaces", "my server"},
		{"special chars", "srv!@#"},
		{"dot", "srv.local"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModel()
			showAddForm(t, m)

			typeString(m, tc.input)
			pressEnter(t, m)

			// Should still be on Name field.
			if m.hostForm.Username != "" {
				t.Errorf("Username = %q, want empty — name %q should be invalid", m.hostForm.Username, tc.input)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// E2E: Edit Host
// ---------------------------------------------------------------------------

func TestE2E_EditHost_FullFlow(t *testing.T) {
	m := newTestModel(config.Host{Name: "dev", Username: "old", Host: "old.local", Port: 22})
	showEditForm(t, m, 0)

	// Clear existing name and type new one.
	clearField(m, len(m.hostForm.Name))
	typeString(m, "staging")
	pressEnter(t, m)

	clearField(m, len(m.hostForm.Username))
	typeString(m, "new")
	pressEnter(t, m)

	clearField(m, len(m.hostForm.Host))
	typeString(m, "new.local")
	pressEnter(t, m)

	// Keep port as-is.
	submitAddForm(t, m)

	if m.screen != screenHostList {
		t.Fatalf("screen = %d, want screenHostList after edit", m.screen)
	}
	h := m.config.Hosts[0]
	if h.Name != "staging" {
		t.Errorf("Name = %q, want %q", h.Name, "staging")
	}
	if h.Username != "new" {
		t.Errorf("Username = %q, want %q", h.Username, "new")
	}
	if h.Host != "new.local" {
		t.Errorf("Host = %q, want %q", h.Host, "new.local")
	}
}

func TestE2E_EditHost_CancelEscape(t *testing.T) {
	original := config.Host{Name: "dev", Username: "alice", Host: "10.0.0.1", Port: 22}
	m := newTestModel(original)
	showEditForm(t, m, 0)

	// Change something.
	clearField(m, 3)
	typeString(m, "changed")
	pressEsc(m)

	if m.screen != screenHostList {
		t.Errorf("screen = %d, want screenHostList after esc", m.screen)
	}
	if m.config.Hosts[0].Name != "dev" {
		t.Errorf("Name = %q, want %q — edit cancel should not change config", m.config.Hosts[0].Name, "dev")
	}
}

func TestE2E_EditHost_PrefilledValues(t *testing.T) {
	original := config.Host{Name: "staging", Username: "deploy", Host: "stage.local", Port: 2222}
	m := newTestModel(original)
	showEditForm(t, m, 0)

	if m.hostForm.Name != "staging" {
		t.Errorf("hostForm.Name = %q, want %q", m.hostForm.Name, "staging")
	}
	if m.hostForm.Username != "deploy" {
		t.Errorf("hostForm.Username = %q, want %q", m.hostForm.Username, "deploy")
	}
	if m.hostForm.Host != "stage.local" {
		t.Errorf("hostForm.Host = %q, want %q", m.hostForm.Host, "stage.local")
	}
	if m.hostForm.Port != "2222" {
		t.Errorf("hostForm.Port = %q, want %q", m.hostForm.Port, "2222")
	}
}

// clearField sends n backspace keystrokes to clear the current field.
func clearField(m *model, n int) {
	for i := 0; i < n; i++ {
		m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	}
}

// ---------------------------------------------------------------------------
// E2E: Delete Host
// ---------------------------------------------------------------------------

func TestE2E_DeleteHost_ConfirmYes(t *testing.T) {
	m := newTestModel(
		config.Host{Name: "a", Username: "u", Host: "h1", Port: 22},
		config.Host{Name: "b", Username: "u", Host: "h2", Port: 22},
	)
	m.showDeleteConfirm(0)

	pressKey(m, 'y')

	if m.screen != screenHostList {
		t.Errorf("screen = %d, want screenHostList after y", m.screen)
	}
	if len(m.config.Hosts) != 1 {
		t.Fatalf("len(Hosts) = %d, want 1", len(m.config.Hosts))
	}
	if m.config.Hosts[0].Name != "b" {
		t.Errorf("remaining host Name = %q, want %q", m.config.Hosts[0].Name, "b")
	}
}

func TestE2E_DeleteHost_CancelNo(t *testing.T) {
	m := newTestModel(config.Host{Name: "a", Username: "u", Host: "h1", Port: 22})
	m.showDeleteConfirm(0)

	pressKey(m, 'n')

	if m.screen != screenHostList {
		t.Errorf("screen = %d, want screenHostList after n", m.screen)
	}
	if len(m.config.Hosts) != 1 {
		t.Errorf("len(Hosts) = %d, want 1 after cancel", len(m.config.Hosts))
	}
}

func TestE2E_DeleteHost_CancelEsc(t *testing.T) {
	m := newTestModel(config.Host{Name: "a", Username: "u", Host: "h1", Port: 22})
	m.showDeleteConfirm(0)

	pressEsc(m)

	if m.screen != screenHostList {
		t.Errorf("screen = %d, want screenHostList after esc", m.screen)
	}
	if len(m.config.Hosts) != 1 {
		t.Errorf("len(Hosts) = %d, want 1 after cancel", len(m.config.Hosts))
	}
}

func TestE2E_DeleteHost_LastHost(t *testing.T) {
	m := newTestModel(config.Host{Name: "only", Username: "u", Host: "h", Port: 22})
	m.showDeleteConfirm(0)

	pressKey(m, 'y')

	if len(m.config.Hosts) != 0 {
		t.Errorf("len(Hosts) = %d, want 0 after deleting last host", len(m.config.Hosts))
	}
}

func TestE2E_DeleteHost_IgnoresOtherKeys(t *testing.T) {
	m := newTestModel(config.Host{Name: "a", Username: "u", Host: "h1", Port: 22})
	m.showDeleteConfirm(0)

	pressKey(m, 'x')

	if m.screen != screenDeleteConfirm {
		t.Errorf("screen = %d, want screenDeleteConfirm — random key should not dismiss", m.screen)
	}
}

// ---------------------------------------------------------------------------
// E2E: Host List Navigation
// ---------------------------------------------------------------------------

func TestE2E_HostList_DownUp(t *testing.T) {
	m := newTestModel(
		config.Host{Name: "a", Username: "u", Host: "h1", Port: 22},
		config.Host{Name: "b", Username: "u", Host: "h2", Port: 22},
		config.Host{Name: "c", Username: "u", Host: "h3", Port: 22},
	)

	// Down through all items (3 hosts + 1 "Add" = 4 items).
	pressDown(m)
	if m.selected != 1 {
		t.Errorf("selected = %d, want 1", m.selected)
	}
	pressDown(m)
	if m.selected != 2 {
		t.Errorf("selected = %d, want 2", m.selected)
	}
	pressDown(m)
	if m.selected != 3 {
		t.Errorf("selected = %d, want 3 (Add item)", m.selected)
	}
	// Clamp at bottom.
	pressDown(m)
	if m.selected != 3 {
		t.Errorf("selected = %d, want 3 (clamped)", m.selected)
	}

	// Up back to top.
	pressUp(m)
	pressUp(m)
	pressUp(m)
	if m.selected != 0 {
		t.Errorf("selected = %d, want 0", m.selected)
	}
	// Clamp at top.
	pressUp(m)
	if m.selected != 0 {
		t.Errorf("selected = %d, want 0 (clamped)", m.selected)
	}
}

func TestE2E_HostList_QuickConnect(t *testing.T) {
	m := newTestModel(
		config.Host{Name: "a", Username: "u", Host: "h1", Port: 22},
		config.Host{Name: "b", Username: "u", Host: "h2", Port: 22},
		config.Host{Name: "c", Username: "u", Host: "h3", Port: 22},
	)

	// Quick-connect to host 2.
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	if cmd == nil {
		t.Error("expected command for quick-connect 2")
	}
}

func TestE2E_HostList_QuickConnect_OutOfRange(t *testing.T) {
	m := newTestModel(
		config.Host{Name: "a", Username: "u", Host: "h1", Port: 22},
	)

	// Quick-connect 5 with only 1 host.
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}})
	if cmd != nil {
		t.Error("quick-connect 5 with 1 host should be no-op")
	}
}

func TestE2E_HostList_EnterOnHost_TriggersConnect(t *testing.T) {
	m := newTestModel(config.Host{Name: "dev", Username: "u", Host: "h", Port: 22})

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("enter on host should produce connect command")
	}
}

func TestE2E_HostList_EnterOnAddItem_OpensForm(t *testing.T) {
	m := newTestModel(config.Host{Name: "dev", Username: "u", Host: "h", Port: 22})
	m.selected = 1 // "Add" item

	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.screen != screenAddHost {
		t.Errorf("screen = %d, want screenAddHost", m.screen)
	}
}

func TestE2E_HostList_SpaceOnHost_TriggersConnect(t *testing.T) {
	m := newTestModel(config.Host{Name: "dev", Username: "u", Host: "h", Port: 22})

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if cmd == nil {
		t.Error("space on host should produce connect command")
	}
}

func TestE2E_HostList_EmptyList_OnlyAddItem(t *testing.T) {
	m := newTestModel()

	// Should show "No hosts registered yet" message.
	viewContains(t, m, "No hosts")

	// Only item is "Add".
	pressDown(m)
	if m.selected != 0 {
		t.Errorf("selected = %d, want 0 (clamped on empty list)", m.selected)
	}

	// Enter on "Add" item opens form.
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pumpCmds(t, m, cmd)
	if m.screen != screenAddHost {
		t.Errorf("screen = %d, want screenAddHost", m.screen)
	}
}

func TestE2E_HostList_KeysEandDOnAddItem_NoOp(t *testing.T) {
	m := newTestModel(config.Host{Name: "dev", Username: "u", Host: "h", Port: 22})
	m.selected = 1 // "Add" item

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if m.screen != screenHostList {
		t.Error("e on Add item should be no-op")
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if m.screen != screenHostList {
		t.Error("d on Add item should be no-op")
	}
}

// ---------------------------------------------------------------------------
// E2E: Screen transitions — round trips
// ---------------------------------------------------------------------------

func TestE2E_Transition_HostList_Add_Esc_HostList(t *testing.T) {
	m := newTestModel()

	// 'a' opens add screen.
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	pumpCmds(t, m, cmd)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if m.screen != screenAddHost {
		t.Fatalf("screen = %d, want screenAddHost", m.screen)
	}

	// Esc returns to host list.
	pressEsc(m)
	if m.screen != screenHostList {
		t.Errorf("screen = %d, want screenHostList after esc", m.screen)
	}
}

func TestE2E_Transition_HostList_Add_Submit_HostList(t *testing.T) {
	m := newTestModel()
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	pumpCmds(t, m, cmd)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	fillAddForm(t, m, "srv", "alice", "h", "22")
	submitAddForm(t, m)

	if m.screen != screenHostList {
		t.Errorf("screen = %d, want screenHostList after submit", m.screen)
	}
	if len(m.config.Hosts) != 1 {
		t.Fatalf("len(Hosts) = %d, want 1", len(m.config.Hosts))
	}
}

func TestE2E_Transition_HostList_Edit_Esc_HostList(t *testing.T) {
	m := newTestModel(config.Host{Name: "dev", Username: "u", Host: "h", Port: 22})

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	pumpCmds(t, m, cmd)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if m.screen != screenEditHost {
		t.Fatalf("screen = %d, want screenEditHost", m.screen)
	}

	pressEsc(m)
	if m.screen != screenHostList {
		t.Errorf("screen = %d, want screenHostList after esc", m.screen)
	}
}

func TestE2E_Transition_DeleteConfirm_CancelRoundTrips(t *testing.T) {
	m := newTestModel(
		config.Host{Name: "a", Username: "u", Host: "h1", Port: 22},
		config.Host{Name: "b", Username: "u", Host: "h2", Port: 22},
	)

	// Open delete via 'd'.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if m.screen != screenDeleteConfirm {
		t.Fatalf("screen = %d, want screenDeleteConfirm", m.screen)
	}

	// Cancel with 'n'.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if m.screen != screenHostList {
		t.Errorf("screen = %d, want screenHostList after n", m.screen)
	}

	// Open again.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if m.screen != screenDeleteConfirm {
		t.Fatalf("screen = %d, want screenDeleteConfirm (2nd time)", m.screen)
	}

	// Cancel with Esc.
	pressEsc(m)
	if m.screen != screenHostList {
		t.Errorf("screen = %d, want screenHostList after esc", m.screen)
	}
}

func TestE2E_Transition_ErrorScreen_AnyKey(t *testing.T) {
	m := newTestModel()
	m.screen = screenError
	m.errMsg = "something broke"

	// Any key returns to host list.
	pressKey(m, 'x')
	if m.screen != screenHostList {
		t.Errorf("screen = %d, want screenHostList after key on error", m.screen)
	}
}

// ---------------------------------------------------------------------------
// E2E: Connect — select host and quit
// ---------------------------------------------------------------------------

func TestE2E_Connect_SelectHostAndQuit(t *testing.T) {
	m := newTestModel(config.Host{Name: "dev", Username: "u", Host: "h", Port: 22})

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter on host should produce quit command")
	}

	// connectIndex should be set.
	if m.connectIndex != 0 {
		t.Errorf("connectIndex = %d, want 0", m.connectIndex)
	}

	// SelectedHost should return the host.
	host, ok := SelectedHost(m)
	if !ok {
		t.Fatal("SelectedHost should return true")
	}
	if host.Name != "dev" {
		t.Errorf("host.Name = %q, want %q", host.Name, "dev")
	}
}

func TestE2E_Connect_QuickConnect(t *testing.T) {
	m := newTestModel(
		config.Host{Name: "a", Username: "u", Host: "h1", Port: 22},
		config.Host{Name: "b", Username: "u", Host: "h2", Port: 22},
	)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	if cmd == nil {
		t.Fatal("quick-connect 2 should produce quit command")
	}

	if m.connectIndex != 1 {
		t.Errorf("connectIndex = %d, want 1", m.connectIndex)
	}

	host, ok := SelectedHost(m)
	if !ok {
		t.Fatal("SelectedHost should return true")
	}
	if host.Name != "b" {
		t.Errorf("host.Name = %q, want %q", host.Name, "b")
	}
}

// ---------------------------------------------------------------------------
// E2E: Window resize
// ---------------------------------------------------------------------------

func TestE2E_WindowResize_OnForm(t *testing.T) {
	m := newTestModel()
	showAddForm(t, m)

	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if m.width != 120 {
		t.Errorf("width = %d, want 120", m.width)
	}
	if m.height != 40 {
		t.Errorf("height = %d, want 40", m.height)
	}
}

// ---------------------------------------------------------------------------
// E2E: View rendering — visibility checks
// ---------------------------------------------------------------------------

func TestE2E_View_HostListShowsHosts(t *testing.T) {
	m := newTestModel(
		config.Host{Name: "alpha", Username: "alice", Host: "10.0.0.1", Port: 22},
		config.Host{Name: "beta", Username: "bob", Host: "10.0.0.2", Port: 2222},
	)
	viewContains(t, m, "alpha", "alice", "10.0.0.1", "beta", "bob", "10.0.0.2:2222")
}

func TestE2E_View_EmptyHostList(t *testing.T) {
	m := newTestModel()
	viewContains(t, m, "No hosts", "Add new host")
}

func TestE2E_View_AddFormShowsFieldTitles(t *testing.T) {
	m := newTestModel()
	showAddForm(t, m)
	viewContains(t, m, "Name", "Username", "Host", "Port")
}

func TestE2E_View_EditFormShowsTitle(t *testing.T) {
	m := newTestModel(config.Host{Name: "dev", Username: "u", Host: "h", Port: 22})
	showEditForm(t, m, 0)
	viewContains(t, m, "Edit Host")
}

func TestE2E_View_AddFormShowsTitle(t *testing.T) {
	m := newTestModel()
	showAddForm(t, m)
	viewContains(t, m, "Add Host")
}

func TestE2E_View_DeleteConfirmShowsHostName(t *testing.T) {
	m := newTestModel(config.Host{Name: "production", Username: "u", Host: "h", Port: 22})
	m.showDeleteConfirm(0)
	viewContains(t, m, "production", "Delete")
}

func TestE2E_View_ErrorScreenShowsMessage(t *testing.T) {
	m := newTestModel()
	m.screen = screenError
	m.errMsg = "connection refused"
	viewContains(t, m, "Error", "connection refused", "Press any key")
}

func TestE2E_View_HostListHelpBar(t *testing.T) {
	m := newTestModel(
		config.Host{Name: "a", Username: "u", Host: "h1", Port: 22},
	)
	viewContains(t, m, "a add", "e edit", "d delete", "q quit")
}

func TestE2E_View_HostListWithUpdate(t *testing.T) {
	m := newTestModel(config.Host{Name: "a", Username: "u", Host: "h1", Port: 22})
	m.updateTag = "v0.2.0"
	viewContains(t, m, "v0.2.0", "update")
}

func TestE2E_View_UpdateConfirm(t *testing.T) {
	m := newTestModel(config.Host{Name: "a", Username: "u", Host: "h1", Port: 22})
	m.updateTag = "v0.2.0"
	m.screen = screenUpdateConfirm
	viewContains(t, m, "v0.2.0", "Update now")
}

func TestE2E_View_FormEscHelp(t *testing.T) {
	m := newTestModel()
	showAddForm(t, m)
	viewContains(t, m, "esc cancel")
}

// ---------------------------------------------------------------------------
// E2E: Search
// ---------------------------------------------------------------------------

func TestE2E_Search_ActivateDeactivate(t *testing.T) {
	m := newTestModel(
		config.Host{Name: "alpha", Username: "u", Host: "h1", Port: 22},
		config.Host{Name: "beta", Username: "u", Host: "h2", Port: 22},
	)
	if m.searchActive {
		t.Error("search should start inactive")
	}

	// Activate with '/'.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !m.searchActive {
		t.Error("search should be active after /")
	}
	viewContains(t, m, "/>")

	// Type query (one rune at a time, like real keyboard input).
	typeString(m, "alp")
	if m.searchQuery != "alp" {
		t.Errorf("searchQuery = %q, want %q", m.searchQuery, "alp")
	}

	// Esc deactivates.
	pressEsc(m)
	if m.searchActive {
		t.Error("search should be inactive after esc")
	}
}

func TestE2E_Search_FiltersHosts(t *testing.T) {
	m := newTestModel(
		config.Host{Name: "alpha", Username: "u", Host: "h1", Port: 22},
		config.Host{Name: "beta", Username: "u", Host: "h2", Port: 22},
		config.Host{Name: "gamma", Username: "u", Host: "h3", Port: 22},
	)

	// Activate and type "alp" — should match "alpha".
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	typeString(m, "alp")
	viewContains(t, m, "alpha")
	viewNotContains(t, m, "beta", "gamma")
}

func TestE2E_Search_NoMatches(t *testing.T) {
	m := newTestModel(
		config.Host{Name: "alpha", Username: "u", Host: "h1", Port: 22},
	)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	typeString(m, "zzz")
	viewContains(t, m, "No matches")
}

func TestE2E_Search_BackspaceDeletesQuery(t *testing.T) {
	m := newTestModel(
		config.Host{Name: "alpha", Username: "u", Host: "h1", Port: 22},
	)
	m.activateSearch()
	m.searchQuery = "ab"
	m.updateFilter()

	m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.searchQuery != "a" {
		t.Errorf("searchQuery = %q, want %q", m.searchQuery, "a")
	}

	m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.searchQuery != "" {
		t.Errorf("searchQuery = %q, want empty", m.searchQuery)
	}

	// One more backspace on empty query should deactivate search.
	m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.searchActive {
		t.Error("search should deactivate after backspace on empty query")
	}
}

func TestE2E_Search_QuitFromSearch(t *testing.T) {
	m := newTestModel(config.Host{Name: "a", Username: "u", Host: "h", Port: 22})
	m.activateSearch()

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Error("q from search should produce quit command")
	}
}

func TestE2E_Search_ActivateWithS(t *testing.T) {
	m := newTestModel(config.Host{Name: "a", Username: "u", Host: "h", Port: 22})

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if !m.searchActive {
		t.Error("s should activate search")
	}
}

// ---------------------------------------------------------------------------
// E2E: Update flow
// ---------------------------------------------------------------------------

func TestE2E_Update_UKeyNoUpdate(t *testing.T) {
	m := newTestModel(config.Host{Name: "a", Username: "u", Host: "h", Port: 22})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if m.screen != screenHostList {
		t.Error("u with no update should stay on host list")
	}
}

func TestE2E_Update_UKeyWithUpdate(t *testing.T) {
	m := newTestModel(config.Host{Name: "a", Username: "u", Host: "h", Port: 22})
	m.latestRelease = &updater.Release{Tag: "v99.0.0"}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if m.screen != screenUpdateConfirm {
		t.Errorf("screen = %d, want screenUpdateConfirm", m.screen)
	}
}

// ---------------------------------------------------------------------------
// E2E: Edge cases
// ---------------------------------------------------------------------------

func TestE2E_Edge_NilMsg_NoOp(t *testing.T) {
	m := newTestModel()
	result, _ := m.Update(nil)
	if result != m {
		t.Error("nil msg should return same model")
	}
}

func TestE2E_Edge_DeleteConfirm_IgnoresRandomKeys(t *testing.T) {
	m := newTestModel(config.Host{Name: "a", Username: "u", Host: "h", Port: 22})
	m.showDeleteConfirm(0)

	for _, key := range []rune{'a', 'b', 'c', '1', '!', 'z'} {
		pressKey(m, key)
		if m.screen != screenDeleteConfirm {
			t.Errorf("key %q should not dismiss delete confirm", string(key))
		}
	}
}

func TestE2E_Edge_UpdateConfirmYesNo(t *testing.T) {
	m := newTestModel(config.Host{Name: "a", Username: "u", Host: "h", Port: 22})
	m.latestRelease = &updater.Release{Tag: "v99.0.0"}
	m.screen = screenUpdateConfirm

	// 'n' cancels.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if m.screen != screenHostList {
		t.Error("n on update confirm should return to host list")
	}

	// Re-enter and try 'y'.
	m.screen = screenUpdateConfirm
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Error("y on update confirm should return a command")
	}

	// Esc cancels.
	m.screen = screenUpdateConfirm
	pressEsc(m)
	if m.screen != screenHostList {
		t.Error("esc on update confirm should return to host list")
	}
}

func TestE2E_Edge_AddHostWithManyHosts(t *testing.T) {
	hosts := make([]config.Host, 9)
	for i := range hosts {
		hosts[i] = config.Host{
			Name:     fmt.Sprintf("host%d", i),
			Username: "u",
			Host:     fmt.Sprintf("h%d", i),
			Port:     22,
		}
	}
	m := newTestModel(hosts...)
	if len(m.config.Hosts) != 9 {
		t.Fatalf("len(Hosts) = %d, want 9", len(m.config.Hosts))
	}

	showAddForm(t, m)
	fillAddForm(t, m, "host10", "u", "h10", "22")
	submitAddForm(t, m)

	if len(m.config.Hosts) != 10 {
		t.Fatalf("len(Hosts) = %d, want 10", len(m.config.Hosts))
	}
}

func TestE2E_Edge_QuitFromHostList(t *testing.T) {
	m := newTestModel()
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Error("q should produce quit command")
	}
}

func TestE2E_Edge_CtrlC_QuitFromHostList(t *testing.T) {
	m := newTestModel()
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Error("ctrl+c should produce quit command")
	}
}

// ---------------------------------------------------------------------------
// E2E: Combo — full CRUD lifecycle
// ---------------------------------------------------------------------------

func TestE2E_Lifecycle_AddConnectEditDelete(t *testing.T) {
	// Start with empty list.
	m := newTestModel()
	if len(m.config.Hosts) != 0 {
		t.Fatalf("expected empty host list, got %d hosts", len(m.config.Hosts))
	}

	// --- ADD ---
	showAddForm(t, m)
	fillAddForm(t, m, "dev", "alice", "192.168.1.1", "22")
	submitAddForm(t, m)

	if len(m.config.Hosts) != 1 {
		t.Fatalf("after add: len(Hosts) = %d, want 1", len(m.config.Hosts))
	}
	if m.config.Hosts[0].Name != "dev" {
		t.Errorf("Name = %q, want %q", m.config.Hosts[0].Name, "dev")
	}

	// --- CONNECT: enter selects host and quits. ---
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter on host should produce quit command")
	}

	// Verify host was selected.
	host, ok := SelectedHost(m)
	if !ok {
		t.Fatal("SelectedHost should return true")
	}
	if host.Name != "dev" {
		t.Errorf("host.Name = %q, want %q", host.Name, "dev")
	}

	// Re-create model for remaining lifecycle (edit + delete).
	m = newTestModel(config.Host{Name: "dev", Username: "alice", Host: "192.168.1.1", Port: 22})

	// --- EDIT ---
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	pumpCmds(t, m, cmd)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if m.screen != screenEditHost {
		t.Fatalf("screen = %d, want screenEditHost", m.screen)
	}

	// Keep name, change username, keep host, keep port.
	pressEnter(t, m) // advance past Name
	clearField(m, len(m.hostForm.Username))
	typeString(m, "bob")
	pressEnter(t, m)    // advance past Username
	pressEnter(t, m)    // advance past Host
	submitAddForm(t, m) // submit on Port

	if m.screen != screenHostList {
		t.Fatalf("screen = %d, want screenHostList after edit", m.screen)
	}
	if m.config.Hosts[0].Username != "bob" {
		t.Errorf("Username = %q, want %q after edit", m.config.Hosts[0].Username, "bob")
	}

	// --- DELETE ---
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if m.screen != screenDeleteConfirm {
		t.Fatalf("screen = %d, want screenDeleteConfirm", m.screen)
	}
	pressKey(m, 'y')
	if len(m.config.Hosts) != 0 {
		t.Fatalf("after delete: len(Hosts) = %d, want 0", len(m.config.Hosts))
	}
}

// ---------------------------------------------------------------------------
// E2E: Combo — multiple hosts navigation
// ---------------------------------------------------------------------------

func TestE2E_MultiHost_NavigateAndOperate(t *testing.T) {
	m := newTestModel(
		config.Host{Name: "alpha", Username: "alice", Host: "h1", Port: 22},
		config.Host{Name: "beta", Username: "bob", Host: "h2", Port: 2222},
		config.Host{Name: "gamma", Username: "carol", Host: "h3", Port: 22},
	)

	// Navigate to beta (index 1).
	pressDown(m)
	if m.selected != 1 {
		t.Errorf("selected = %d, want 1", m.selected)
	}

	// Edit beta.
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	pumpCmds(t, m, cmd)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if m.screen != screenEditHost {
		t.Fatalf("screen = %d, want screenEditHost", m.screen)
	}
	// Verify pre-filled.
	if m.hostForm.Name != "beta" {
		t.Errorf("prefilled Name = %q, want %q", m.hostForm.Name, "beta")
	}
	if m.hostForm.Username != "bob" {
		t.Errorf("prefilled Username = %q, want %q", m.hostForm.Username, "bob")
	}

	// Cancel edit.
	pressEsc(m)
	if m.screen != screenHostList {
		t.Errorf("screen = %d, want screenHostList", m.screen)
	}

	// Navigate to gamma (index 2).
	pressDown(m)
	if m.selected != 2 {
		t.Errorf("selected = %d, want 2", m.selected)
	}

	// Delete gamma.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	pressKey(m, 'y')
	if len(m.config.Hosts) != 2 {
		t.Errorf("len(Hosts) = %d, want 2 after delete", len(m.config.Hosts))
	}

	// Verify remaining hosts.
	names := make(map[string]bool)
	for _, h := range m.config.Hosts {
		names[h.Name] = true
	}
	if names["gamma"] {
		t.Error("gamma should have been deleted")
	}
	if !names["alpha"] || !names["beta"] {
		t.Error("alpha and beta should still exist")
	}
}
