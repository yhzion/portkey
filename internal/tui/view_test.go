package tui

import (
	"strings"
	"testing"
)

// view tests lock rendering behavior so style refactors don't change output structure.

func TestView_HostList_Empty(t *testing.T) {
	m := newTestModel()
	view := m.View()

	if !strings.Contains(view, "Portkey") {
		t.Error("empty host list should show title")
	}
	if !strings.Contains(view, "Add new host") {
		t.Error("empty host list should show add item")
	}
	if !strings.Contains(view, "No hosts registered") {
		t.Error("empty host list should show empty message")
	}
}

func TestView_HostList_WithHosts(t *testing.T) {
	m := newTestModel(testHostDev)
	view := m.View()

	if !strings.Contains(view, "Portkey") {
		t.Error("host list should show title")
	}
	if !strings.Contains(view, "dev") {
		t.Error("host list should show display name")
	}
	if !strings.Contains(view, "u@h") {
		t.Error("host list should show connection info")
	}
	if !strings.Contains(view, "Add new host") {
		t.Error("host list should show add item")
	}
	if !strings.Contains(view, "↑") {
		t.Error("host list should show help bar")
	}
}

func TestView_HostList_CustomPort(t *testing.T) {
	m := newTestModel(testHostPort)
	view := m.View()

	if !strings.Contains(view, ":2222") {
		t.Error("custom port should appear in connection info")
	}
}

func TestView_AddHost(t *testing.T) {
	m := newTestModel()
	m.showAddScreen()
	view := m.View()

	if !strings.Contains(view, "Add Host") {
		t.Error("add screen should show 'Add Host'")
	}
	if !strings.Contains(view, "esc") {
		t.Error("add screen should show cancel help")
	}
}

func TestView_EditHost(t *testing.T) {
	m := newTestModel(testHostDev)
	m.showEditScreen(0)
	view := m.View()

	if !strings.Contains(view, "Edit Host") {
		t.Error("edit screen should show 'Edit Host'")
	}
}

func TestView_DeleteConfirm(t *testing.T) {
	m := newTestModel(testHostDev)
	m.showDeleteConfirm(0)
	view := m.View()

	if !strings.Contains(view, "Delete") {
		t.Error("delete confirm should show 'Delete'")
	}
	if !strings.Contains(view, "dev") {
		t.Error("delete confirm should show host name")
	}
	if !strings.Contains(view, "[y]") || !strings.Contains(view, "[n]") {
		t.Error("delete confirm should show y/n options")
	}
}

func TestView_Error(t *testing.T) {
	m := newTestModel()
	m.screen = screenError
	m.errMsg = "connection failed"
	view := m.View()

	if !strings.Contains(view, "Error") {
		t.Error("error screen should show 'Error'")
	}
	if !strings.Contains(view, "connection failed") {
		t.Error("error screen should show error message")
	}
	if !strings.Contains(view, "any key") {
		t.Error("error screen should show return hint")
	}
}

func TestView_SelectedItemHasCursor(t *testing.T) {
	m := newTestModel(testHostDev, testHostA)
	view := m.View()

	if !strings.Contains(view, "▸") {
		t.Error("selected item should show cursor")
	}
}

func TestView_DefaultPortNotShown(t *testing.T) {
	m := newTestModel(testHostDev)
	view := m.View()

	if strings.Contains(view, ":22") {
		t.Error("default port 22 should not appear in connection info")
	}
}
