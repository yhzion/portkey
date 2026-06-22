package tui

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yhzion/portkey/internal/updater"
)

// fakeInstaller is a test double for the Installer interface that records
// whether DownloadAndInstall was called and with which release.
type fakeInstaller struct {
	called bool
	gotRel *updater.Release
	err    error
}

func (f *fakeInstaller) DownloadAndInstall(rel *updater.Release, _ func(string)) error {
	f.called = true
	f.gotRel = rel
	return f.err
}

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
	m.updateModel.latestRelease = &updater.Release{Tag: "v99.0.0"}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if m.screen != screenUpdateConfirm {
		t.Errorf("screen = %d, want screenUpdateConfirm", m.screen)
	}
}

func TestUpdateConfirm_Y(t *testing.T) {
	m := newTestModel(testHostDev)
	fi := &fakeInstaller{}
	m.updateModel.installer = fi
	m.updateModel.latestRelease = &updater.Release{Tag: "v99.0.0"}
	m.screen = screenUpdateConfirm

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatal("y on update confirm should return a command")
	}

	// Execute the command and verify the installer was actually called.
	msg := cmd()
	if !fi.called {
		t.Error("installer.DownloadAndInstall was not called")
	}
	if fi.gotRel == nil || fi.gotRel.Tag != "v99.0.0" {
		t.Errorf("installer called with wrong release: %v", fi.gotRel)
	}
	done, ok := msg.(updateDoneMsg)
	if !ok {
		t.Fatalf("cmd() returned %T, want updateDoneMsg", msg)
	}
	if done.err != nil {
		t.Errorf("updateDoneMsg.err = %v, want nil", done.err)
	}

	// Feed the success message back and verify the notification screen.
	m2, _ := m.Update(done)
	mm := m2.(*model)
	if mm.screen != screenNotification {
		t.Errorf("screen = %d, want screenNotification after success", mm.screen)
	}
	if mm.errMsg == "" {
		t.Error("success notification message should be set")
	}
}

func TestUpdateConfirm_Y_InstallError(t *testing.T) {
	m := newTestModel(testHostDev)
	installErr := errors.New("disk full")
	fi := &fakeInstaller{err: installErr}
	m.updateModel.installer = fi
	m.updateModel.latestRelease = &updater.Release{Tag: "v99.0.0"}
	m.screen = screenUpdateConfirm

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatal("y on update confirm should return a command")
	}

	msg := cmd()
	done, ok := msg.(updateDoneMsg)
	if !ok {
		t.Fatalf("cmd() returned %T, want updateDoneMsg", msg)
	}
	if done.err == nil {
		t.Fatal("updateDoneMsg.err should be non-nil on install failure")
	}

	// Feed the error message back and verify the error screen.
	m2, _ := m.Update(done)
	mm := m2.(*model)
	if mm.screen != screenError {
		t.Errorf("screen = %d, want screenError after install failure", mm.screen)
	}
	if mm.errMsg == "" {
		t.Error("error message should be set")
	}
	// The returned error must wrap the original installer error.
	if !errors.Is(done.err, installErr) {
		t.Errorf("error chain does not contain original error: %v", done.err)
	}
}

func TestUpdateConfirm_Y_NoInstaller(t *testing.T) {
	// When no installer is configured, pressing y must return an error, not succeed silently.
	m := newTestModel(testHostDev)
	m.updateModel.installer = nil
	m.updateModel.latestRelease = &updater.Release{Tag: "v99.0.0"}
	m.screen = screenUpdateConfirm

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatal("y on update confirm should return a command")
	}

	msg := cmd()
	done, ok := msg.(updateDoneMsg)
	if !ok {
		t.Fatalf("cmd() returned %T, want updateDoneMsg", msg)
	}
	if done.err == nil {
		t.Error("updateDoneMsg.err should be non-nil when no installer is configured")
	}
}

func TestUpdateConfirm_N(t *testing.T) {
	m := newTestModel(testHostDev)
	m.updateModel.latestRelease = &updater.Release{Tag: "v99.0.0"}
	m.screen = screenUpdateConfirm
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if m.screen != screenHostList {
		t.Error("n on update confirm should return to host list")
	}
}

func TestUpdateConfirm_Esc(t *testing.T) {
	m := newTestModel(testHostDev)
	m.updateModel.latestRelease = &updater.Release{Tag: "v99.0.0"}
	m.screen = screenUpdateConfirm
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.screen != screenHostList {
		t.Error("esc on update confirm should return to host list")
	}
}

func TestUpdateAvailableMsg_SetsNotification(t *testing.T) {
	m := newTestModel(testHostDev)
	m.Update(updateAvailableMsg{Tag: "v0.2.0", Rel: &updater.Release{Tag: "v0.2.0"}})
	if m.updateModel.tag != "v0.2.0" {
		t.Errorf("updateTag = %q, want %q", m.updateModel.tag, "v0.2.0")
	}
	if m.updateModel.latestRelease == nil {
		t.Error("latestRelease should be set")
	}
}

func TestUpdateCheckFailedMsg_Silent(t *testing.T) {
	m := newTestModel(testHostDev)
	m.Update(updateCheckFailedMsg{})
	if m.screen != screenHostList {
		t.Error("update check failure should not change screen")
	}
	if m.updateModel.tag != "" {
		t.Error("update check failure should not set update tag")
	}
}
