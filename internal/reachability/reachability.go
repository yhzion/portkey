package reachability

import (
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// Reachability represents the network reachability status of a host.
type Reachability int

const (
	StatusUnknown Reachability = iota // "..."
	StatusUp                          // "UP"
	StatusDown                        // "DOWN"
)

// String implements fmt.Stringer. Used for display and logging.
func (r Reachability) String() string {
	switch r {
	case StatusUp:
		return "UP"
	case StatusDown:
		return "DOWN"
	case StatusUnknown:
		return "..."
	default:
		return "..."
	}
}

// MarshalJSON implements json.Marshaler. Serializes as a string, not an int.
func (r Reachability) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.String())
}

// UnmarshalJSON implements json.Unmarshaler. Parses string back to enum.
func (r *Reachability) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	switch s {
	case "UP":
		*r = StatusUp
	case "DOWN":
		*r = StatusDown
	default:
		*r = StatusUnknown
	}
	return nil
}

// Check performs a TCP dial check against host:port with a 3-second timeout.
// Returns StatusUp if the connection succeeds, StatusDown otherwise.
// A port of 0 defaults to 22.
func Check(host string, port int) Reachability {
	if port == 0 {
		port = 22
	}
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return StatusDown
	}
	conn.Close()
	return StatusUp
}
