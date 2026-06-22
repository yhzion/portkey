package netfix

import (
	"context"
	"errors"
	"net"
	"testing"
)

// dialRecorder builds a fake dialFunc that fails for any address in failOn and
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

func TestFallbackDial_PrimarySucceeds_NoFallback(t *testing.T) {
	dial, seen := dialRecorder(map[string]bool{})
	wrapped := fallbackDial(dial, []string{"1.1.1.1:53"})

	conn, err := wrapped(context.Background(), "udp", "9.9.9.9:53")
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	conn.Close()
	if len(*seen) != 1 || (*seen)[0] != "9.9.9.9:53" {
		t.Fatalf("expected only the primary to be dialed, got %v", *seen)
	}
}

func TestFallbackDial_PrimaryFails_UsesFallback(t *testing.T) {
	// Mimics Termux: the resolver-provided [::1]:53 is refused.
	dial, seen := dialRecorder(map[string]bool{"[::1]:53": true})
	wrapped := fallbackDial(dial, []string{"1.1.1.1:53", "8.8.8.8:53"})

	conn, err := wrapped(context.Background(), "udp", "[::1]:53")
	if err != nil {
		t.Fatalf("expected fallback success, got %v", err)
	}
	conn.Close()
	want := []string{"[::1]:53", "1.1.1.1:53"}
	if len(*seen) != len(want) || (*seen)[0] != want[0] || (*seen)[1] != want[1] {
		t.Fatalf("expected primary then first fallback, got %v", *seen)
	}
}

func TestFallbackDial_AllFail_ReturnsOriginalError(t *testing.T) {
	dial, seen := dialRecorder(map[string]bool{
		"[::1]:53":   true,
		"1.1.1.1:53": true,
		"8.8.8.8:53": true,
	})
	wrapped := fallbackDial(dial, []string{"1.1.1.1:53", "8.8.8.8:53"})

	_, err := wrapped(context.Background(), "udp", "[::1]:53")
	if err == nil {
		t.Fatal("expected error when every server fails")
	}
	if len(*seen) != 3 {
		t.Fatalf("expected all three servers to be tried, got %v", *seen)
	}
}
