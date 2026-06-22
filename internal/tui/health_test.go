package tui

import (
	"errors"
	"net"
	"strconv"
	"testing"
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
