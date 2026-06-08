package tui

import (
	"errors"
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

	msg := m.checkUpdate()()

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

	if msg := m.checkUpdate()(); msg != nil {
		t.Errorf("checkUpdate() up-to-date msg = %v, want nil", msg)
	}
}

func TestCheckUpdate_Error(t *testing.T) {
	checker := fakeUpdateChecker{err: errors.New("network down")}
	m := InitialModel(&config.Config{}, "v0.1.0", checker, mockStore{}).(*model)

	if _, ok := m.checkUpdate()().(updateCheckFailedMsg); !ok {
		t.Error("expected updateCheckFailedMsg on error")
	}
}
