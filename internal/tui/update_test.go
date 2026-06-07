package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Shared fixtures live in model_test.go (same internal package).

// --- Screen transitions ---

func TestShowAddScreen_SetsState(t *testing.T) {
	m := newTestModel()
	m.showAddScreen()
	if m.screen != screenAddHost {
		t.Errorf("screen = %d, want screenAddHost", m.screen)
	}
	if m.hostForm == nil {
		t.Error("hostForm should not be nil")
	}
	if m.hostForm.Port != testPort22 {
		t.Errorf("default Port = %q, want %q", m.hostForm.Port, testPort22)
	}
}

func TestShowEditScreen_PrefillsForm(t *testing.T) {
	m := newTestModel(testHostPort)
	m.showEditScreen(0)
	if m.screen != screenEditHost {
		t.Errorf("screen = %d, want screenEditHost", m.screen)
	}
	if m.editIndex != 0 {
		t.Errorf("editIndex = %d, want 0", m.editIndex)
	}
	if m.hostForm.DisplayName != testHostPort.DisplayName {
		t.Errorf("DisplayName = %q, want %q", m.hostForm.DisplayName, testHostPort.DisplayName)
	}
	if m.hostForm.Port != "2222" {
		t.Errorf("Port = %q, want %q", m.hostForm.Port, "2222")
	}
}

func TestShowDeleteConfirm_SetsState(t *testing.T) {
	m := newTestModel(testHostDev)
	m.showDeleteConfirm(0)
	if m.screen != screenDeleteConfirm {
		t.Errorf("screen = %d, want screenDeleteConfirm", m.screen)
	}
	if m.editIndex != 0 {
		t.Errorf("editIndex = %d, want 0", m.editIndex)
	}
}

// --- HostList key handling ---

func TestHostList_UpClampsAtZero(t *testing.T) {
	m := newTestModel(testHostA, testHostB)
	upModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if upModel.(*model).selected != 0 {
		t.Error("selected should stay 0 at top")
	}
}

func TestHostList_Down(t *testing.T) {
	m := newTestModel(testHostA, testHostB)
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.selected != 1 {
		t.Errorf("selected = %d, want 1 after down", m.selected)
	}
}

func TestHostList_Up(t *testing.T) {
	m := newTestModel(testHostA, testHostB)
	m.selected = 1
	m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.selected != 0 {
		t.Errorf("selected = %d, want 0 after up", m.selected)
	}
}

func TestHostList_DownClampsAtLast(t *testing.T) {
	m := newTestModel(testHostA)
	// 1 host + "Add" item = 2 total, index 1 is last
	m.selected = 1
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.selected != 1 {
		t.Error("selected should stay at last item")
	}
}

func TestHostList_Quit(t *testing.T) {
	m := newTestModel()
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Error("expected quit command, got nil")
	}
}

func TestHostList_AKey_OpensAddScreen(t *testing.T) {
	m := newTestModel()
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if m.screen != screenAddHost {
		t.Errorf("screen = %d, want screenAddHost after 'a'", m.screen)
	}
}

func TestHostList_EKeyOnHost_OpensEdit(t *testing.T) {
	m := newTestModel(testHostDev)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if m.screen != screenEditHost {
		t.Errorf("screen = %d, want screenEditHost after 'e' on host", m.screen)
	}
}

func TestHostList_EKeyOnAddItem_NoOp(t *testing.T) {
	m := newTestModel(testHostDev)
	m.selected = 1 // "Add" item
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if m.screen != screenHostList {
		t.Error("e on Add item should be no-op")
	}
}

func TestHostList_DKeyOnHost_OpensDeleteConfirm(t *testing.T) {
	m := newTestModel(testHostDev)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if m.screen != screenDeleteConfirm {
		t.Errorf("screen = %d, want screenDeleteConfirm after 'd'", m.screen)
	}
}

func TestHostList_DKeyOnAddItem_NoOp(t *testing.T) {
	m := newTestModel(testHostDev)
	m.selected = 1
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if m.screen != screenHostList {
		t.Error("d on Add item should be no-op")
	}
}

func TestHostList_EnterOnHost_TriggersConnect(t *testing.T) {
	m := newTestModel(testHostDev)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("expected a command from enter on host, got nil")
	}
}

func TestHostList_EnterOnAddItem_OpensAdd(t *testing.T) {
	m := newTestModel(testHostDev)
	m.selected = 1
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != screenAddHost {
		t.Error("enter on Add item should open add screen")
	}
}

func TestHostList_SpaceOnHost_TriggersConnect(t *testing.T) {
	m := newTestModel(testHostDev)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if cmd == nil {
		t.Error("expected a command from space on host, got nil")
	}
}

func TestHostList_QuickConnect_ValidNumber(t *testing.T) {
	m := newTestModel(testHostA, testHostB, testHostC)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	if cmd == nil {
		t.Error("expected command for quick-connect 2")
	}
}

func TestHostList_QuickConnect_InvalidNumber_NoOp(t *testing.T) {
	m := newTestModel(testHostA)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}})
	if cmd != nil {
		t.Error("quick-connect 5 with only 1 host should be no-op")
	}
}

// --- Delete confirm ---

func TestDeleteConfirm_Y(t *testing.T) {
	m := newTestModel(testHostA, testHostB)
	m.showDeleteConfirm(0)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if m.screen != screenHostList {
		t.Error("after y, should return to host list")
	}
	if len(m.config.Hosts) != 1 {
		t.Errorf("len(Hosts) = %d, want 1 after delete", len(m.config.Hosts))
	}
}

func TestDeleteConfirm_N(t *testing.T) {
	m := newTestModel(testHostA)
	m.showDeleteConfirm(0)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if m.screen != screenHostList {
		t.Error("after n, should return to host list")
	}
	if len(m.config.Hosts) != 1 {
		t.Error("after n, host should still exist")
	}
}

func TestDeleteConfirm_Esc(t *testing.T) {
	m := newTestModel(testHostA)
	m.showDeleteConfirm(0)
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.screen != screenHostList {
		t.Error("after esc, should return to host list")
	}
}

func TestDeleteConfirm_SingleHostBecomesEmpty(t *testing.T) {
	m := newTestModel(testHostDev)
	m.showDeleteConfirm(0)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if len(m.config.Hosts) != 0 {
		t.Errorf("len(Hosts) = %d, want 0 after deleting last host", len(m.config.Hosts))
	}
}

// --- Error screen ---

func TestErrorScreen_AnyKey(t *testing.T) {
	m := newTestModel()
	m.screen = screenError
	m.errMsg = "test error"
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if m.screen != screenHostList {
		t.Error("any key on error screen should return to host list")
	}
}

// --- Window resize ---

func TestUpdate_WindowSizeMsg(t *testing.T) {
	m := newTestModel()
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	if m.width != 100 {
		t.Errorf("width = %d, want 100", m.width)
	}
	if m.height != 40 {
		t.Errorf("height = %d, want 40", m.height)
	}
}

func TestUpdate_NilMsg_NoOp(t *testing.T) {
	m := newTestModel()
	result, _ := m.Update(nil)
	if result != m {
		t.Error("nil msg should return same model")
	}
}

// --- SSH done message ---

func TestUpdate_SshDoneSuccess(t *testing.T) {
	m := newTestModel()
	m.screen = screenError
	m.Update(sshDoneMsg{err: nil})
	if m.screen != screenHostList {
		t.Error("on SSH success, should go to host list")
	}
}

func TestUpdate_SshDoneError(t *testing.T) {
	m := newTestModel()
	m.Update(sshDoneMsg{err: errTest})
	if m.screen != screenError {
		t.Error("on SSH error, should go to error screen")
	}
	if m.errMsg == "" {
		t.Error("errMsg should be set on SSH error")
	}
}

// --- Empty host list ---

func TestEmptyHostList_NavigationClamp(t *testing.T) {
	m := newTestModel()
	if m.selected != 0 {
		t.Errorf("selected = %d, want 0 for empty list", m.selected)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.selected != 0 {
		t.Error("should not move down past single Add item in empty list")
	}
}

func TestHostList_FormEscape(t *testing.T) {
	m := newTestModel()
	m.showAddScreen()
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.screen != screenHostList {
		t.Error("esc on form should return to host list")
	}
}

// errTest is a sentinel for tests that need a non-nil error.
var errTest = &testError{}

type testError struct{}

func (e *testError) Error() string { return "test error" }
