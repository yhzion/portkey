package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yhzion/portkey/internal/updater"
)

// --- Update flow tests ---

func TestHostList_UKey_NoUpdateAvailable(t *testing.T) {
	m := newTestModel(testHostDev)
	// No update available — should be no-op
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if m.screen != screenHostList {
		t.Error("u with no update should stay on host list")
	}
}

func TestHostList_UKey_WithUpdateAvailable(t *testing.T) {
	m := newTestModel(testHostDev)
	m.latestRelease = &updater.Release{Tag: "v99.0.0"}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if m.screen != screenUpdateConfirm {
		t.Errorf("screen = %d, want screenUpdateConfirm", m.screen)
	}
}

func TestUpdateConfirm_Y(t *testing.T) {
	m := newTestModel(testHostDev)
	m.latestRelease = &updater.Release{Tag: "v99.0.0"}
	m.screen = screenUpdateConfirm
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Error("y on update confirm should return a command")
	}
}

func TestUpdateConfirm_N(t *testing.T) {
	m := newTestModel(testHostDev)
	m.latestRelease = &updater.Release{Tag: "v99.0.0"}
	m.screen = screenUpdateConfirm
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if m.screen != screenHostList {
		t.Error("n on update confirm should return to host list")
	}
}

func TestUpdateConfirm_Esc(t *testing.T) {
	m := newTestModel(testHostDev)
	m.latestRelease = &updater.Release{Tag: "v99.0.0"}
	m.screen = screenUpdateConfirm
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.screen != screenHostList {
		t.Error("esc on update confirm should return to host list")
	}
}

func TestUpdateAvailableMsg_SetsNotification(t *testing.T) {
	m := newTestModel(testHostDev)
	m.Update(updateAvailableMsg{Tag: "v0.2.0", Rel: &updater.Release{Tag: "v0.2.0"}})
	if m.updateTag != "v0.2.0" {
		t.Errorf("updateTag = %q, want %q", m.updateTag, "v0.2.0")
	}
	if m.latestRelease == nil {
		t.Error("latestRelease should be set")
	}
}

func TestUpdateCheckFailedMsg_Silent(t *testing.T) {
	m := newTestModel(testHostDev)
	m.Update(updateCheckFailedMsg{})
	if m.screen != screenHostList {
		t.Error("update check failure should not change screen")
	}
	if m.updateTag != "" {
		t.Error("update check failure should not set update tag")
	}
}
