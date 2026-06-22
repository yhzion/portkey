package tui

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yhzion/portkey/internal/config"
	"github.com/yhzion/portkey/internal/updater"
)

// blockingChecker is a fake UpdateChecker that blocks until its context is
// cancelled, simulating a slow in-flight network check.
type blockingChecker struct {
	// started is closed when CheckLatest begins executing (so tests can
	// synchronise the cancel call to after the check has started).
	started chan struct{}
}

func newBlockingChecker() *blockingChecker {
	return &blockingChecker{started: make(chan struct{})}
}

func (b *blockingChecker) CheckLatest(ctx context.Context) (*updater.Release, error) {
	close(b.started)
	<-ctx.Done()
	return nil, ctx.Err()
}

// TestCheckUpdate_ContextCancelled_NoFailedMsg asserts that when checkUpdate's
// context is cancelled (user quitting), it returns nil rather than an
// updateCheckFailedMsg — so no "offline" hint is flashed.
func TestCheckUpdate_ContextCancelled_NoFailedMsg(t *testing.T) {
	checker := newBlockingChecker()
	m := InitialModel(&config.Config{}, "v0.1.0", checker, mockStore{}).(*model)

	ctx, cancel := context.WithCancel(context.Background())
	cmd := m.updateModel.checkUpdate(ctx)

	done := make(chan tea.Msg, 1)
	go func() {
		done <- cmd()
	}()

	// Wait until the checker has started, then cancel.
	<-checker.started
	cancel()

	select {
	case msg := <-done:
		if msg != nil {
			t.Errorf("checkUpdate() after context cancel = %T(%v), want nil", msg, msg)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("checkUpdate() did not return promptly after context cancel")
	}
}

// TestCancelFunc_InvokedOnQuit verifies that when the host-list quit key is
// pressed, the model's update-check cancel func is invoked.
func TestCancelFunc_InvokedOnQuit(t *testing.T) {
	checker := newBlockingChecker()
	cfg := &config.Config{}
	m := InitialModel(cfg, "v0.1.0", checker, mockStore{}).(*model)

	cancelled := false
	m.updateModel.cancelCheck = func() {
		cancelled = true
	}

	// Press 'q' to quit.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	if !cancelled {
		t.Error("quit key should have invoked cancelCheck func")
	}
}
