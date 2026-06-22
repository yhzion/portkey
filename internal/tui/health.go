package tui

import (
	"errors"
	"net"
)

// healthStatus is a host's reachability as last observed.
type healthStatus int

const (
	healthUnknown healthStatus = iota // not yet checked, checking, or DNS-unresolvable
	healthOnline                      // TCP connect to the SSH port succeeded
	healthOffline                     // connection refused / timed out
)

// classifyDialError maps a net.DialTimeout result to a healthStatus. A DNS
// resolution failure is treated as unknown rather than offline so SSH-config
// aliases (which don't resolve via DNS) aren't shown as down.
func classifyDialError(err error) healthStatus {
	if err == nil {
		return healthOnline
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return healthUnknown
	}
	return healthOffline
}
