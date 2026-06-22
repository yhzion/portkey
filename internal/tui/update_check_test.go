package tui

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/yhzion/portkey/internal/config"
	"github.com/yhzion/portkey/internal/updater"
)

// fakeUpdateChecker stands in for a real updater.Client so the update-check
// flow can be exercised without any network access.
type fakeUpdateChecker struct {
	rel *updater.Release
	err error
}

func (f fakeUpdateChecker) CheckLatest() (*updater.Release, error) {
	return f.rel, f.err
}

func TestCheckUpdate_NewerAvailable(t *testing.T) {
	checker := fakeUpdateChecker{rel: &updater.Release{Tag: "v99.0.0"}}
	m := InitialModel(&config.Config{}, "v0.1.0", checker, mockStore{}).(*model)

	msg := m.updateModel.checkUpdate()()

	avail, ok := msg.(updateAvailableMsg)
	if !ok {
		t.Fatalf("checkUpdate() msg = %T, want updateAvailableMsg", msg)
	}
	if avail.Tag != "v99.0.0" {
		t.Errorf("tag = %q, want v99.0.0", avail.Tag)
	}
}

func TestCheckUpdate_UpToDate(t *testing.T) {
	checker := fakeUpdateChecker{rel: &updater.Release{Tag: "v0.1.0"}}
	m := InitialModel(&config.Config{}, "v0.1.0", checker, mockStore{}).(*model)

	if msg := m.updateModel.checkUpdate()(); msg != nil {
		t.Errorf("checkUpdate() up-to-date msg = %v, want nil", msg)
	}
}

func TestCheckUpdate_Error(t *testing.T) {
	checker := fakeUpdateChecker{err: errors.New("network down")}
	m := InitialModel(&config.Config{}, "v0.1.0", checker, mockStore{}).(*model)

	if _, ok := m.updateModel.checkUpdate()().(updateCheckFailedMsg); !ok {
		t.Error("expected updateCheckFailedMsg on error")
	}
}

// TestCheckUpdate_ErrorCarriesKind verifies that checkUpdate emits an
// updateCheckFailedMsg carrying a non-zero Kind when the checker fails.
func TestCheckUpdate_ErrorCarriesKind(t *testing.T) {
	checker := fakeUpdateChecker{err: errors.New("network down")}
	m := InitialModel(&config.Config{}, "v0.1.0", checker, mockStore{}).(*model)

	msg, ok := m.updateModel.checkUpdate()().(updateCheckFailedMsg)
	if !ok {
		t.Fatal("expected updateCheckFailedMsg, got other type")
	}
	// Kind must not be zero (unset) — any non-zero kind is acceptable for a
	// generic error; exact kind is tested per-error-type below.
	if msg.Kind == updater.KindUnknown {
		t.Error("updateCheckFailedMsg.Kind should be set (non-zero), got KindUnknown")
	}
}

// TestCheckUpdate_KindClassification verifies that checkUpdate correctly
// classifies common error kinds from the updater into updateCheckFailedMsg.
func TestCheckUpdate_KindClassification(t *testing.T) {
	tests := []struct {
		name     string
		checker  fakeUpdateChecker
		wantKind updater.CheckErrorKind
	}{
		{
			"offline",
			fakeUpdateChecker{err: fmt.Errorf("fetch latest release: %w", errors.New("connection refused"))},
			updater.KindOffline,
		},
		{
			"rate_limited_429",
			fakeUpdateChecker{err: fmt.Errorf("rate limited")},
			updater.KindRateLimited,
		},
		{
			"not_found_404",
			fakeUpdateChecker{err: fmt.Errorf("no releases published yet")},
			updater.KindNotFound,
		},
		{
			"other_403",
			fakeUpdateChecker{err: fmt.Errorf("forbidden: %w", fmt.Errorf("forbidden"))},
			updater.KindOther,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := InitialModel(&config.Config{}, "v0.1.0", tt.checker, mockStore{}).(*model)
			msg, ok := m.updateModel.checkUpdate()().(updateCheckFailedMsg)
			if !ok {
				t.Fatal("expected updateCheckFailedMsg")
			}
			if msg.Kind != tt.wantKind {
				t.Errorf("Kind = %v, want %v", msg.Kind, tt.wantKind)
			}
		})
	}
}

// TestUpdateCheckFailed_NonFatal verifies that receiving updateCheckFailedMsg
// keeps the model on screenHostList and does not trigger screenError.
func TestUpdateCheckFailed_NonFatal(t *testing.T) {
	m := newTestModel(testHostDev)
	result, _ := m.Update(updateCheckFailedMsg{Kind: updater.KindOffline})
	next := result.(*model)
	if next.screen == screenError {
		t.Error("updateCheckFailedMsg must not route to screenError")
	}
	if next.screen != screenHostList {
		t.Errorf("screen = %v, want screenHostList after updateCheckFailedMsg", next.screen)
	}
}

// TestView_UpdateCheckFailed_Hints verifies that each failure kind renders
// a distinct quiet hint in the host-list view.
func TestView_UpdateCheckFailed_Hints(t *testing.T) {
	tests := []struct {
		kind     updater.CheckErrorKind
		wantHint string
	}{
		{updater.KindOffline, "offline"},
		{updater.KindRateLimited, "rate-limited"},
		{updater.KindNotFound, "no releases"},
		{updater.KindOther, "update check failed"},
	}

	for _, tt := range tests {
		t.Run(tt.kind.String(), func(t *testing.T) {
			m := newTestModel(testHostDev)
			m.updateModel.checkFailKind = tt.kind
			view := m.View()
			if !strings.Contains(strings.ToLower(view), strings.ToLower(tt.wantHint)) {
				t.Errorf("view does not contain hint %q for kind %v\nview:\n%s", tt.wantHint, tt.kind, view)
			}
		})
	}
}
