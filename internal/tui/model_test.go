package tui

import (
	"errors"
	"testing"

	"github.com/yhzion/portkey/internal/config"
)

// Shared test fixtures — single source of truth.

const (
	testUsername = "alice"
	testHost     = "server"
	testPort22   = "22"
)

var (
	testHostDev  = config.Host{Name: "dev", Username: "u", Host: "h", Port: 22}
	testHostA    = config.Host{Name: "a", Username: "u", Host: "h1", Port: 22}
	testHostB    = config.Host{Name: "b", Username: "u", Host: "h2", Port: 22}
	testHostC    = config.Host{Name: "c", Username: "u", Host: "h3", Port: 22}
	testHostPort = config.Host{Name: "staging", Username: "u", Host: "h", Port: 2222}
)

// mockStore is a no-op Store for tests.
type mockStore struct{}

func (mockStore) Load() (*config.Config, error) { return nil, nil }
func (mockStore) Save(_ *config.Config) error   { return nil }

func newBaseForm() *hostForm {
	return &hostForm{Username: testUsername, Host: testHost, Port: testPort22}
}

func newTestModel(hosts ...config.Host) *model {
	cfg := &config.Config{Hosts: hosts}
	return InitialModel(cfg, "v0.1.0", nil, nil, mockStore{}).(*model)
}

func TestHostForm_ToHost_DefaultPort(t *testing.T) {
	h := newBaseForm().toHost()
	if h.Port != 22 {
		t.Errorf("Port = %d, want 22", h.Port)
	}
}

func TestHostForm_ToHost_CustomPort(t *testing.T) {
	f := newBaseForm()
	f.Port = "2222"
	h := f.toHost()
	if h.Port != 2222 {
		t.Errorf("Port = %d, want 2222", h.Port)
	}
}

func TestHostForm_ToHost_PortOne(t *testing.T) {
	f := newBaseForm()
	f.Port = "1"
	h := f.toHost()
	if h.Port != 1 {
		t.Errorf("Port = %d, want 1", h.Port)
	}
}

func TestHostForm_ToHost_Port65535(t *testing.T) {
	f := newBaseForm()
	f.Port = "65535"
	h := f.toHost()
	if h.Port != 65535 {
		t.Errorf("Port = %d, want 65535", h.Port)
	}
}

func TestHostForm_ToHost_InvalidPortFallsBackTo22(t *testing.T) {
	f := newBaseForm()
	f.Port = "abc"
	h := f.toHost()
	if h.Port != 22 {
		t.Errorf("Port = %d, want 22 for invalid input", h.Port)
	}
}

func TestHostForm_ToHost_EmptyPortFallsBackTo22(t *testing.T) {
	f := newBaseForm()
	f.Port = ""
	h := f.toHost()
	if h.Port != 22 {
		t.Errorf("Port = %d, want 22 when empty", h.Port)
	}
}

func TestParsePort(t *testing.T) {
	tests := []struct {
		input   string
		want    int
		wantErr bool
	}{
		{"", 22, false},
		{"22", 22, false},
		{"8080", 8080, false},
		{"1", 1, false},
		{"65535", 65535, false},
		{"0", 0, true},
		{"65536", 0, true},
		{"abc", 0, true},
		{"22abc", 0, true},
		{" 22 ", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parsePort(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePort(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parsePort(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestHostForm_ToHost_NameFallback(t *testing.T) {
	h := newBaseForm().toHost()
	if h.Name != testUsername {
		t.Errorf("Name = %q, want %q (fallback to username)", h.Name, testUsername)
	}
}

func TestHostForm_ToHost_NameSet(t *testing.T) {
	f := newBaseForm()
	f.Name = "my-server"
	h := f.toHost()
	if h.Name != "my-server" {
		t.Errorf("Name = %q, want %q", h.Name, "my-server")
	}
}

func TestInitialModel_SetsDefaults(t *testing.T) {
	cfg := &config.Config{Hosts: []config.Host{}}
	m := InitialModel(cfg, "v0.1.0", nil, nil, mockStore{}).(*model)

	if m.screen != screenHostList {
		t.Errorf("screen = %d, want screenHostList", m.screen)
	}
	if m.selected != 0 {
		t.Errorf("selected = %d, want 0", m.selected)
	}
	if m.config != cfg {
		t.Error("config pointer mismatch")
	}
}

func TestInitialModel_WithExistingHosts(t *testing.T) {
	cfg := &config.Config{Hosts: []config.Host{testHostDev, testHostB}}
	m := InitialModel(cfg, "v0.1.0", nil, nil, mockStore{}).(*model)
	if len(m.config.Hosts) != 2 {
		t.Errorf("len(Hosts) = %d, want 2", len(m.config.Hosts))
	}
}

// TestSaveAndGoBack_NoDataRace_SaveInFlightAddHost verifies that firing the
// save cmd returned by saveAndGoBack in a goroutine does not race with a
// concurrent AddHost on the main model. Under the old code, the save closure
// captured m.config (not a snapshot), so json.MarshalIndent inside Save read
// the same Hosts slice that AddHost appends to — detectable by -race.
func TestSaveAndGoBack_NoDataRace_SaveInFlightAddHost(t *testing.T) {
	m := newTestModel(testHostA, testHostB)
	m.screen = screenAddHost
	m.formModel.hostForm = &hostForm{Username: "u", Host: "h", Port: "22"}

	cmd := m.saveAndGoBack()

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = cmd() // runs in goroutine, reads config via Save
	}()

	// Mutate the same slice concurrently while Save reads it.
	m.config.AddHost(testHostC)
	m.config.RemoveHost(0)

	<-done
}

// TestSaveAndGoBack_NoDataRace_ConcurrentSaves verifies that two save
// commands fired back-to-back do not race against each other's reads of the
// shared config.
func TestSaveAndGoBack_NoDataRace_ConcurrentSaves(t *testing.T) {
	m := newTestModel(testHostA, testHostB)
	m.screen = screenAddHost
	m.formModel.hostForm = &hostForm{Username: "u", Host: "h", Port: "22"}

	cmd1 := m.saveAndGoBack()
	m.screen = screenEditHost
	m.editIndex = 0
	cmd2 := m.saveAndGoBack()

	done := make(chan struct{}, 2)
	go func() { _ = cmd1(); done <- struct{}{} }()
	go func() { _ = cmd2(); done <- struct{}{} }()
	<-done
	<-done
}

// TestSaveAndGoBack_RollbackOnSaveFailure asserts that when the async Save
// fails, the in-memory config reverts to its pre-mutation state. Without
// rollback, the add leaves a phantom host that is not on disk.
func TestSaveAndGoBack_RollbackOnSaveFailure(t *testing.T) {
	saveErr := errors.New("disk full")
	cfg := &config.Config{Hosts: []config.Host{testHostA}}
	m := InitialModel(cfg, "v0.1.0", nil, nil, failingStore{err: saveErr}).(*model)
	m.screen = screenAddHost
	m.formModel.hostForm = &hostForm{Username: "admin", Host: "10.0.0.1", Port: "22"}

	beforeHosts := len(m.config.Hosts)

	msg := m.saveAndGoBack()()

	if _, ok := msg.(errMsg); !ok {
		t.Fatalf("saveAndGoBack() on store error = %T, want errMsg", msg)
	}
	if len(m.config.Hosts) != beforeHosts {
		t.Errorf("after failed save, len(Hosts) = %d, want %d (rollback)", len(m.config.Hosts), beforeHosts)
	}
}

// TestConfirmDelete_RollbackOnSaveFailure asserts that a failed delete save
// restores the removed host so the UI matches disk.
func TestConfirmDelete_RollbackOnSaveFailure(t *testing.T) {
	saveErr := errors.New("read-only fs")
	cfg := &config.Config{Hosts: []config.Host{testHostA, testHostB}}
	m := InitialModel(cfg, "v0.1.0", nil, nil, failingStore{err: saveErr}).(*model)
	m.showDeleteConfirm(0)

	beforeHosts := len(m.config.Hosts)

	msg := m.confirmDelete()()

	if _, ok := msg.(errMsg); !ok {
		t.Fatalf("confirmDelete() on store error = %T, want errMsg", msg)
	}
	if len(m.config.Hosts) != beforeHosts {
		t.Errorf("after failed delete save, len(Hosts) = %d, want %d (rollback)", len(m.config.Hosts), beforeHosts)
	}
	if m.config.Hosts[0] != testHostA {
		t.Errorf("Hosts[0] = %+v, want %+v (restored)", m.config.Hosts[0], testHostA)
	}
}
