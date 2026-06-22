package tui

import (
	"errors"
	"net"
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
