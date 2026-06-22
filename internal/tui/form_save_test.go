package tui

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yhzion/portkey/internal/config"
)

// failingStore is a Store whose Save always fails, exercising the error path
// that the no-op mockStore can't reach.
type failingStore struct{ err error }

func (s failingStore) Load() (*config.Config, error) { return nil, s.err }
func (s failingStore) Save(_ *config.Config) error   { return s.err }

func TestSaveAndGoBack_SaveError(t *testing.T) {
	saveErr := errors.New("disk full")
	m := InitialModel(&config.Config{}, "v0.1.0", nil, nil, failingStore{err: saveErr}).(*model)
	m.screen = screenAddHost
	m.formModel.hostForm = &hostForm{Username: "admin", Host: "10.0.0.1", Port: "22"}

	msg := m.saveAndGoBack()()

	em, ok := msg.(errMsg)
	if !ok {
		t.Fatalf("saveAndGoBack() on store error = %T, want errMsg", msg)
	}
	if !errors.Is(em.err, saveErr) {
		t.Errorf("errMsg wraps %v, want %v", em.err, saveErr)
	}
}

func TestSaveAndGoBack_SaveSuccess(t *testing.T) {
	m := InitialModel(&config.Config{}, "v0.1.0", nil, nil, mockStore{}).(*model)
	m.screen = screenAddHost
	m.formModel.hostForm = &hostForm{Username: "admin", Host: "10.0.0.1", Port: "22"}

	if msg := m.saveAndGoBack()(); msg != nil {
		t.Errorf("saveAndGoBack() on success = %v, want nil", msg)
	}
	if m.screen != screenHostList {
		t.Errorf("screen = %d, want screenHostList after save", m.screen)
	}
}

// forwardToForm should delegate to the embedded form and, when the form is not
// complete, keep the model on the form screen.
func TestForwardToForm_NoActiveForm(t *testing.T) {
	m := InitialModel(&config.Config{}, "v0.1.0", nil, nil, mockStore{}).(*model)
	m.screen = screenAddHost // no form built yet

	got, cmd := m.forwardToForm(tea.WindowSizeMsg{Width: 80, Height: 24})

	if got.(*model) != m {
		t.Error("forwardToForm should return the same model")
	}
	if cmd != nil {
		t.Errorf("expected nil cmd when no form is active, got %T", cmd)
	}
	if m.screen != screenAddHost {
		t.Errorf("screen = %d, want screenAddHost", m.screen)
	}
}

func TestForwardToForm_ActiveFormStaysOnScreen(t *testing.T) {
	m := InitialModel(&config.Config{}, "v0.1.0", nil, nil, mockStore{}).(*model)
	m.screen = screenAddHost
	m.formModel.showAdd()

	m.forwardToForm(tea.WindowSizeMsg{Width: 80, Height: 24})

	if m.screen != screenAddHost {
		t.Errorf("screen = %d, want screenAddHost (incomplete form)", m.screen)
	}
}
