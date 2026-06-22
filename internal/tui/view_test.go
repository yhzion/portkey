package tui

import (
	"strings"
	"testing"

	"github.com/yhzion/portkey/internal/updater"
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

func TestView_Notification(t *testing.T) {
	m := newTestModel()
	m.screen = screenNotification
	m.errMsg = "Update successful. Please restart portkey."
	view := m.View()

	if strings.Contains(view, "Error") {
		t.Error("notification screen should NOT show 'Error' prefix")
	}
	if !strings.Contains(view, "Update successful") {
		t.Error("notification screen should show message")
	}
	if !strings.Contains(view, "any key") {
		t.Error("notification screen should show return hint")
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

func TestView_UpdateNotification(t *testing.T) {
	m := newTestModel(testHostDev)
	m.updateModel.tag = "v0.2.0"
	view := m.View()

	if !strings.Contains(view, "available") {
		t.Error("should show update notification when updateTag is set")
	}
	if !strings.Contains(view, "v0.2.0") {
		t.Error("should show version in update notification")
	}
}

func TestView_UpdateConfirm(t *testing.T) {
	m := newTestModel(testHostDev)
	m.updateModel.tag = "v0.2.0"
	m.updateModel.latestRelease = &updater.Release{Tag: "v0.2.0"}
	m.screen = screenUpdateConfirm
	view := m.View()

	if !strings.Contains(view, "Update") {
		t.Error("update confirm should show Update")
	}
	if !strings.Contains(view, "v0.2.0") {
		t.Error("update confirm should show target version")
	}
	if !strings.Contains(view, "[y]") {
		t.Error("update confirm should show y/n options")
	}
}

func TestView_NoUpdateNotification(t *testing.T) {
	m := newTestModel(testHostDev)
	view := m.View()

	if strings.Contains(view, "available") {
		t.Error("should not show update notification when no update")
	}
}

// TestView_HelpBar_UpdateHint_Hidden asserts that "u update" is absent from
// the populated-list help bar when no update is cached (tag == "").
func TestView_HelpBar_UpdateHint_Hidden(t *testing.T) {
	m := newTestModel(testHostDev)
	// tag is "" by default — no update available
	view := m.View()

	if strings.Contains(view, "u update") {
		t.Error("help bar must NOT contain 'u update' when no update is available")
	}
}

// TestView_HelpBar_UpdateHint_Shown asserts that "u update" appears in the
// populated-list help bar when an update is cached (tag != "").
func TestView_HelpBar_UpdateHint_Shown(t *testing.T) {
	m := newTestModel(testHostDev)
	m.updateModel.tag = "v1.0.0"
	view := m.View()

	if !strings.Contains(view, "u update") {
		t.Error("help bar must contain 'u update' when an update is available")
	}
}
