package tui

import (
	"errors"
	"net"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yhzion/portkey/internal/config"
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

const healthTimeout = 1500 * time.Millisecond

// healthResultMsg carries one host's check result back into Update.
type healthResultMsg struct {
	name   string
	status healthStatus
}

// dialHost attempts a TCP connection to the host's SSH port. Thin wrapper over
// the standard library; the meaningful logic is in classifyDialError.
func dialHost(host string, port int) error {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), healthTimeout)
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
}

// checkHostCmd returns a Cmd that dials one host in the background and reports
// the result as a healthResultMsg.
func checkHostCmd(name, host string, port int) tea.Cmd {
	return func() tea.Msg {
		return healthResultMsg{name: name, status: classifyDialError(dialHost(host, port))}
	}
}

// healthCheckAll batches a check for every host. Returns nil when there are no
// hosts so Init has nothing to run.
func healthCheckAll(hosts []config.Host) tea.Cmd {
	if len(hosts) == 0 {
		return nil
	}
	cmds := make([]tea.Cmd, 0, len(hosts))
	for _, h := range hosts {
		cmds = append(cmds, checkHostCmd(h.Name, h.Host, h.Port))
	}
	return tea.Batch(cmds...)
}
