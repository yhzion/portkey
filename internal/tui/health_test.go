package tui

import (
	"errors"
	"net"
	"strconv"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yhzion/portkey/internal/config"
)

func TestClassifyDialError(t *testing.T) {
	if got := classifyDialError(nil); got != healthOnline {
		t.Errorf("nil err -> %v, want healthOnline", got)
	}
	dnsErr := &net.DNSError{Err: "no such host", Name: "rtx5090", IsNotFound: true}
	if got := classifyDialError(dnsErr); got != healthUnknown {
		t.Errorf("DNS error -> %v, want healthUnknown", got)
	}
	if got := classifyDialError(errors.New("connection refused")); got != healthOffline {
		t.Errorf("generic error -> %v, want healthOffline", got)
	}
}

func TestCheckHostCmd_OpenPort_Online(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	host, portStr, _ := net.SplitHostPort(ln.Addr().String())
	port, _ := strconv.Atoi(portStr)

	msg := checkHostCmd("alpha", host, port)()
	hr, ok := msg.(healthResultMsg)
	if !ok {
		t.Fatalf("got %T, want healthResultMsg", msg)
	}
	if hr.name != "alpha" || hr.status != healthOnline {
		t.Errorf("got {%q, %v}, want {alpha, healthOnline}", hr.name, hr.status)
	}
}

func TestCheckHostCmd_ClosedPort_Offline(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	host, portStr, _ := net.SplitHostPort(ln.Addr().String())
	port, _ := strconv.Atoi(portStr)
	ln.Close() // nothing listens on this port now

	msg := checkHostCmd("beta", host, port)()
	hr := msg.(healthResultMsg)
	if hr.status != healthOffline {
		t.Errorf("closed port -> %v, want healthOffline", hr.status)
	}
}

func TestHealthCheckAll_EmptyIsNil(t *testing.T) {
	if healthCheckAll(nil) != nil {
		t.Error("healthCheckAll(nil) should be nil so tea.Batch has nothing to do")
	}
}

func TestUpdate_HealthResultMsg_StoresStatus(t *testing.T) {
	m := newTestModel(testHostDev) // testHostDev.Name == "dev"
	updated, _ := m.Update(healthResultMsg{name: "dev", status: healthOnline})
	mm := updated.(*model)
	if mm.health["dev"] != healthOnline {
		t.Errorf("health[dev] = %v, want healthOnline", mm.health["dev"])
	}
}

func TestHandleHostListKey_R_ResetsAndRechecks(t *testing.T) {
	m := newTestModel(testHostDev)
	m.health["dev"] = healthOnline

	_, cmd := m.handleHostListKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})

	if m.health["dev"] != healthUnknown {
		t.Errorf("after r, health[dev] = %v, want healthUnknown (reset)", m.health["dev"])
	}
	if cmd == nil {
		t.Error("r should return a re-check command")
	}
}

func TestNameColor(t *testing.T) {
	if nameColor(healthOnline) != colorPositive {
		t.Errorf("online -> %q, want colorPositive", nameColor(healthOnline))
	}
	if nameColor(healthOffline) != colorDim {
		t.Errorf("offline -> %q, want colorDim", nameColor(healthOffline))
	}
	if nameColor(healthUnknown) != "" {
		t.Errorf("unknown -> %q, want empty (no color)", nameColor(healthUnknown))
	}
}

// A colored name must still render on one line (no layout regression).
func TestRenderHostItem_OnlineStatus_SingleLine(t *testing.T) {
	m := newTestModel()
	h := config.Host{Name: "datamaker-192-168-14-135", Username: "u", Host: "h", Port: 22}
	m.health[h.Name] = healthOnline
	row := m.renderHostItem(0, h, false, nil, nameColumnWidth([]string{h.Name}))
	if body := strings.TrimSuffix(row, "\n"); strings.Contains(body, "\n") {
		t.Errorf("colored row wrapped:\n%q", row)
	}
}
