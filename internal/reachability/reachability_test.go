package reachability_test

import (
	"encoding/json"
	"net"
	"testing"

	"github.com/yhzion/portkey/internal/reachability"
)

// --- Enum String() tests ---

func TestStatusUnknown_String(t *testing.T) {
	got := reachability.StatusUnknown.String()
	want := "..."
	if got != want {
		t.Errorf("StatusUnknown.String() = %q, want %q", got, want)
	}
}

func TestStatusUp_String(t *testing.T) {
	got := reachability.StatusUp.String()
	want := "UP"
	if got != want {
		t.Errorf("StatusUp.String() = %q, want %q", got, want)
	}
}

func TestStatusDown_String(t *testing.T) {
	got := reachability.StatusDown.String()
	want := "DOWN"
	if got != want {
		t.Errorf("StatusDown.String() = %q, want %q", got, want)
	}
}

func TestInvalidReachability_String(t *testing.T) {
	invalid := reachability.Reachability(99)
	got := invalid.String()
	want := "..."
	if got != want {
		t.Errorf("invalid Reachability.String() = %q, want %q", got, want)
	}
}

// --- JSON round-trip tests ---

func TestMarshalJSON_Up(t *testing.T) {
	data, err := json.Marshal(reachability.StatusUp)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `"UP"` {
		t.Errorf("MarshalJSON(StatusUp) = %s, want %q", data, `"UP"`)
	}
}

func TestMarshalJSON_Down(t *testing.T) {
	data, err := json.Marshal(reachability.StatusDown)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `"DOWN"` {
		t.Errorf("MarshalJSON(StatusDown) = %s, want %q", data, `"DOWN"`)
	}
}

func TestMarshalJSON_Unknown(t *testing.T) {
	data, err := json.Marshal(reachability.StatusUnknown)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `"..."` {
		t.Errorf("MarshalJSON(StatusUnknown) = %s, want %q", data, `"..."`)
	}
}

func TestUnmarshalJSON_Up(t *testing.T) {
	var r reachability.Reachability
	if err := json.Unmarshal([]byte(`"UP"`), &r); err != nil {
		t.Fatal(err)
	}
	if r != reachability.StatusUp {
		t.Errorf("got %d, want StatusUp", r)
	}
}

func TestUnmarshalJSON_Down(t *testing.T) {
	var r reachability.Reachability
	if err := json.Unmarshal([]byte(`"DOWN"`), &r); err != nil {
		t.Fatal(err)
	}
	if r != reachability.StatusDown {
		t.Errorf("got %d, want StatusDown", r)
	}
}

func TestUnmarshalJSON_Unknown(t *testing.T) {
	var r reachability.Reachability
	if err := json.Unmarshal([]byte(`"..."`), &r); err != nil {
		t.Fatal(err)
	}
	if r != reachability.StatusUnknown {
		t.Errorf("got %d, want StatusUnknown", r)
	}
}

func TestUnmarshalJSON_InvalidFallsBackToUnknown(t *testing.T) {
	var r reachability.Reachability
	if err := json.Unmarshal([]byte(`"garbage"`), &r); err != nil {
		t.Fatal(err)
	}
	if r != reachability.StatusUnknown {
		t.Errorf("got %d, want StatusUnknown for unrecognized string", r)
	}
}

func TestJSONRoundTrip(t *testing.T) {
	statuses := []reachability.Reachability{
		reachability.StatusUnknown,
		reachability.StatusUp,
		reachability.StatusDown,
	}
	for _, original := range statuses {
		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("Marshal(%v): %v", original, err)
		}
		var decoded reachability.Reachability
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Unmarshal(%s): %v", data, err)
		}
		if decoded != original {
			t.Errorf("round-trip: got %v, want %v", decoded, original)
		}
	}
}

// --- Check() tests with real TCP listener ---

func TestCheck_PortOpen_ReturnsUp(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	addr := ln.Addr().(*net.TCPAddr)
	status := reachability.Check("127.0.0.1", addr.Port)
	if status != reachability.StatusUp {
		t.Errorf("Check() = %v, want StatusUp for open port", status)
	}
}

func TestCheck_PortClosed_ReturnsDown(t *testing.T) {
	// Find a port that's definitely not listening.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().(*net.TCPAddr)
	port := addr.Port
	ln.Close()

	status := reachability.Check("127.0.0.1", port)
	if status != reachability.StatusDown {
		t.Errorf("Check() = %v, want StatusDown for closed port", status)
	}
}

func TestCheck_InvalidHost_ReturnsDown(t *testing.T) {
	status := reachability.Check("256.256.256.256", 22)
	if status != reachability.StatusDown {
		t.Errorf("Check() = %v, want StatusDown for invalid host", status)
	}
}

func TestCheck_ZeroPort_UsesPort22(t *testing.T) {
	// Port 0 should default to 22. We can't guarantee port 22 state,
	// so just verify it doesn't panic and returns a valid status.
	status := reachability.Check("127.0.0.1", 0)
	if status != reachability.StatusUp && status != reachability.StatusDown {
		t.Errorf("Check() with port 0 = %v, want StatusUp or StatusDown", status)
	}
}
