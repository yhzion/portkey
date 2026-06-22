package netfix

import (
	"context"
	"errors"
	"net"
	"testing"
)

// dialRecorder builds a dialFunc that fails for any address in failOn and
// succeeds (returning one end of a net.Pipe) for anything else, recording every
// address it is asked to dial.
func dialRecorder(failOn map[string]bool) (dialFunc, *[]string) {
	var seen []string
	dial := func(ctx context.Context, network, address string) (net.Conn, error) {
		seen = append(seen, address)
		if failOn[address] {
			return nil, errors.New("connection refused")
		}
		c, _ := net.Pipe()
		return c, nil
	}
	return dial, &seen
}

func TestIsLoopbackHostPort(t *testing.T) {
	cases := map[string]bool{
		"127.0.0.1:53": true,
		"[::1]:53":     true,
		"1.1.1.1:53":   false,
		"8.8.8.8:53":   false,
		"10.0.0.1:53":  false, // a real (private) resolv.conf nameserver
		"garbage":      false,
	}
	for addr, want := range cases {
		if got := isLoopbackHostPort(addr); got != want {
			t.Errorf("isLoopbackHostPort(%q) = %v, want %v", addr, got, want)
		}
	}
}

// A loopback nameserver (Termux's dead default) must be redirected to the first
// real server — NOT passed through, because a UDP dial to it would wrongly
// succeed.
func TestRedirectDial_Loopback_RedirectsToRealServer(t *testing.T) {
	dial, seen := dialRecorder(map[string]bool{})
	wrapped := redirectDial(dial, []string{"1.1.1.1:53", "8.8.8.8:53"})

	conn, err := wrapped(context.Background(), "udp", "[::1]:53")
	if err != nil {
		t.Fatalf("expected redirect success, got %v", err)
	}
	conn.Close()
	if len(*seen) != 1 || (*seen)[0] != "1.1.1.1:53" {
		t.Fatalf("expected the loopback dial to be redirected to 1.1.1.1:53, got %v", *seen)
	}
}

// A real nameserver from a valid resolv.conf must pass through untouched.
func TestRedirectDial_RealServer_PassesThrough(t *testing.T) {
	dial, seen := dialRecorder(map[string]bool{})
	wrapped := redirectDial(dial, []string{"1.1.1.1:53"})

	conn, err := wrapped(context.Background(), "udp", "10.0.0.1:53")
	if err != nil {
		t.Fatalf("expected passthrough success, got %v", err)
	}
	conn.Close()
	if len(*seen) != 1 || (*seen)[0] != "10.0.0.1:53" {
		t.Fatalf("expected passthrough to 10.0.0.1:53, got %v", *seen)
	}
}

// When the first real server errors (e.g. TCP connect refused), the next is
// tried in order.
func TestRedirectDial_FirstServerFails_TriesNext(t *testing.T) {
	dial, seen := dialRecorder(map[string]bool{"1.1.1.1:53": true})
	wrapped := redirectDial(dial, []string{"1.1.1.1:53", "8.8.8.8:53"})

	conn, err := wrapped(context.Background(), "tcp", "127.0.0.1:53")
	if err != nil {
		t.Fatalf("expected fallback to second server, got %v", err)
	}
	conn.Close()
	want := []string{"1.1.1.1:53", "8.8.8.8:53"}
	if len(*seen) != 2 || (*seen)[0] != want[0] || (*seen)[1] != want[1] {
		t.Fatalf("expected both servers tried in order, got %v", *seen)
	}
}

func TestRedirectDial_AllServersFail_ReturnsError(t *testing.T) {
	dial, _ := dialRecorder(map[string]bool{"1.1.1.1:53": true, "8.8.8.8:53": true})
	wrapped := redirectDial(dial, []string{"1.1.1.1:53", "8.8.8.8:53"})

	if _, err := wrapped(context.Background(), "tcp", "[::1]:53"); err == nil {
		t.Fatal("expected error when every server fails")
	}
}
