package tui

import (
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
	return InitialModel(cfg, "v0.1.0", nil, mockStore{}).(*model)
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
	m := InitialModel(cfg, "v0.1.0", nil, mockStore{}).(*model)

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
	m := InitialModel(cfg, "v0.1.0", nil, mockStore{}).(*model)
	if len(m.config.Hosts) != 2 {
		t.Errorf("len(Hosts) = %d, want 2", len(m.config.Hosts))
	}
}
