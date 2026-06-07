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
	testHostDev  = config.Host{DisplayName: "dev", Username: "u", Host: "h", Port: 22}
	testHostA    = config.Host{DisplayName: "a", Username: "u", Host: "h1", Port: 22}
	testHostB    = config.Host{DisplayName: "b", Username: "u", Host: "h2", Port: 22}
	testHostC    = config.Host{DisplayName: "c", Username: "u", Host: "h3", Port: 22}
	testHostPort = config.Host{DisplayName: "staging", Username: "u", Host: "h", Port: 2222}
)

func newBaseForm() *hostForm {
	return &hostForm{Username: testUsername, Host: testHost, Port: testPort22}
}

func newTestModel(hosts ...config.Host) *model {
	cfg := &config.Config{Hosts: hosts}
	return InitialModel(cfg, "v0.1.0", nil).(*model)
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

func TestHostForm_ToHost_DisplayNameFallback(t *testing.T) {
	h := newBaseForm().toHost()
	if h.DisplayName != testUsername {
		t.Errorf("DisplayName = %q, want %q (fallback to username)", h.DisplayName, testUsername)
	}
}

func TestHostForm_ToHost_DisplayNameSet(t *testing.T) {
	f := newBaseForm()
	f.DisplayName = "My Server"
	h := f.toHost()
	if h.DisplayName != "My Server" {
		t.Errorf("DisplayName = %q, want %q", h.DisplayName, "My Server")
	}
}

func TestInitialModel_SetsDefaults(t *testing.T) {
	cfg := &config.Config{Hosts: []config.Host{}}
	m := InitialModel(cfg, "v0.1.0", nil).(*model)

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
	m := InitialModel(cfg, "v0.1.0", nil).(*model)
	if len(m.config.Hosts) != 2 {
		t.Errorf("len(Hosts) = %d, want 2", len(m.config.Hosts))
	}
}
