# Host Health Check Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Show each host's reachability in the list by coloring its name (green=online, dim=offline, default=unknown), checked in the background via TCP dial.

**Architecture:** A pure classifier turns a dial result into a status. A Bubbletea `tea.Cmd` per host dials its SSH port in a goroutine and emits a `healthResultMsg`; `Update` records it in a `map[string]healthStatus` on the model. `renderHostItem` (already a `*model` method) reads that map and colors the name. No layout change — only the name's foreground color varies.

**Tech Stack:** Go, Bubbletea (`tea.Cmd`/`tea.Msg`/`tea.Batch`), Lipgloss, standard `net`.

## Global Constraints

- Build is `CGO_ENABLED=0` for all targets incl. android — no cgo-only APIs.
- Reachability uses **TCP dial**, never ICMP (ICMP needs root/`CAP_NET_RAW`, unavailable on Termux non-root).
- Dial timeout: `1500 * time.Millisecond`.
- DNS-resolution failures classify as `healthUnknown`, NOT offline (SSH aliases like `rtx5090` don't resolve via DNS).
- Color constants already exist in `internal/tui/styles.go`: `colorPositive` (green), `colorDim` (gray). Do not introduce new colors.
- Follow the existing async precedent: `updateModel.checkUpdate(ctx) tea.Cmd { return func() tea.Msg {...} }` in `internal/tui/update_model.go`.

---

### Task 1: Health status type + dial-error classifier

**Files:**
- Create: `internal/tui/health.go`
- Test: `internal/tui/health_test.go`

**Interfaces:**
- Produces: `type healthStatus int` with `healthUnknown` (iota 0), `healthOnline`, `healthOffline`; `func classifyDialError(err error) healthStatus`.

- [ ] **Step 1: Write the failing test**

```go
// internal/tui/health_test.go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestClassifyDialError`
Expected: build FAIL — `undefined: classifyDialError`, `undefined: healthOnline`, etc.

- [ ] **Step 3: Write minimal implementation**

```go
// internal/tui/health.go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/ -run TestClassifyDialError -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tui/health.go internal/tui/health_test.go
git commit -m "feat(tui): healthStatus type and dial-error classifier"
```

---

### Task 2: Dial wrapper + per-host check command

**Files:**
- Modify: `internal/tui/health.go`
- Test: `internal/tui/health_test.go`

**Interfaces:**
- Consumes: `classifyDialError`, `healthStatus` (Task 1).
- Produces: `type healthResultMsg struct { name string; status healthStatus }`; `func checkHostCmd(name, host string, port int) tea.Cmd`; `func healthCheckAll(hosts []config.Host) tea.Cmd`; `const healthTimeout = 1500 * time.Millisecond`.

- [ ] **Step 1: Write the failing test**

```go
// add to internal/tui/health_test.go
import (
	"strconv"

	"github.com/yhzion/portkey/internal/config"
)

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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run 'TestCheckHostCmd|TestHealthCheckAll'`
Expected: build FAIL — `undefined: checkHostCmd`, `undefined: healthCheckAll`.

- [ ] **Step 3: Write minimal implementation**

```go
// add to internal/tui/health.go
import (
	"net"        // (already imported)
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yhzion/portkey/internal/config"
)

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
```

Note: merge the new imports into the existing `import (...)` block from Task 1 (`errors`, `net` stay).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/ -run 'TestCheckHostCmd|TestHealthCheckAll' -v`
Expected: PASS (3 tests)

- [ ] **Step 5: Commit**

```bash
git add internal/tui/health.go internal/tui/health_test.go
git commit -m "feat(tui): background per-host TCP health-check command"
```

---

### Task 3: Model state + Update handling

**Files:**
- Modify: `internal/tui/model.go` (struct field ~line 211 area; `InitialModel` ~line 302)
- Modify: `internal/tui/update.go` (`Update` switch ~lines 12-60)
- Test: `internal/tui/health_test.go`

**Interfaces:**
- Consumes: `healthResultMsg`, `healthStatus` (Tasks 1-2).
- Produces: `model.health map[string]healthStatus` populated on `healthResultMsg`.

- [ ] **Step 1: Write the failing test**

```go
// add to internal/tui/health_test.go
func TestUpdate_HealthResultMsg_StoresStatus(t *testing.T) {
	m := newTestModel(testHostDev) // testHostDev.Name == "dev"
	updated, _ := m.Update(healthResultMsg{name: "dev", status: healthOnline})
	mm := updated.(*model)
	if mm.health["dev"] != healthOnline {
		t.Errorf("health[dev] = %v, want healthOnline", mm.health["dev"])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestUpdate_HealthResultMsg`
Expected: FAIL — `m.health` is a nil map / panic on assignment, or field undefined (build error).

- [ ] **Step 3: Write minimal implementation**

In `internal/tui/model.go`, add the field to the `model` struct (near `connectIndex`):

```go
	// Health-check results keyed by host name (absent == healthUnknown).
	health map[string]healthStatus
```

In `InitialModel` (the `m := &model{...}` literal), initialize it:

```go
		health:   map[string]healthStatus{},
```

In `internal/tui/update.go`, add a case to the `Update` type switch (alongside the other `*Msg` cases, e.g. after `case updateDoneMsg:`):

```go
	case healthResultMsg:
		m.health[msg.name] = msg.status
		return m, nil
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/ -run TestUpdate_HealthResultMsg -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tui/model.go internal/tui/update.go internal/tui/health_test.go
git commit -m "feat(tui): store host health results on the model"
```

---

### Task 4: Trigger checks on startup and on `r`

**Files:**
- Modify: `internal/tui/model.go` (`Init` ~line 317)
- Modify: `internal/tui/update.go` (`handleHostListKey` ~lines 81-137)
- Test: `internal/tui/health_test.go`

**Interfaces:**
- Consumes: `healthCheckAll` (Task 2), `model.health` (Task 3).
- Produces: startup batch includes health checks; `r` key clears `model.health` and re-runs checks.

- [ ] **Step 1: Write the failing test**

```go
// add to internal/tui/health_test.go
import tea "github.com/charmbracelet/bubbletea" // merge into existing imports

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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestHandleHostListKey_R`
Expected: FAIL — `r` is unhandled, so health is unchanged and cmd is nil.

- [ ] **Step 3: Write minimal implementation**

In `internal/tui/update.go` `handleHostListKey`, add a case in the main `switch` (next to `case msg.String() == "a":`):

```go
	case msg.String() == "r":
		m.health = map[string]healthStatus{}
		return m, healthCheckAll(m.config.Hosts)
```

In `internal/tui/model.go`, change `Init` to batch the health checks with the update check:

```go
func (m *model) Init() tea.Cmd {
	config.SortHosts(m.config.Hosts)
	var cmds []tea.Cmd
	if m.updateModel.checker != nil && m.updateModel.version != "dev" {
		cmds = append(cmds, m.updateModel.checkUpdate(context.Background()))
	}
	cmds = append(cmds, healthCheckAll(m.config.Hosts))
	return tea.Batch(cmds...)
}
```

(`tea.Batch` ignores nil cmds, so an empty host list is fine.)

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/ -run TestHandleHostListKey_R -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tui/model.go internal/tui/update.go internal/tui/health_test.go
git commit -m "feat(tui): run health checks on startup and on r key"
```

---

### Task 5: Color the host name by status

**Files:**
- Modify: `internal/tui/health.go` (add `nameColor`)
- Modify: `internal/tui/styles.go` (add `styledName`)
- Modify: `internal/tui/view.go` (`renderHostItem` ~line 131)
- Test: `internal/tui/health_test.go`

**Interfaces:**
- Consumes: `healthStatus` (Task 1), `model.health` (Task 3), `nameStyle`/`colorPositive`/`colorDim` (existing).
- Produces: `func nameColor(status healthStatus) string`; `func styledName(status healthStatus, width int) lipgloss.Style`; `renderHostItem` colors the name.

- [ ] **Step 1: Write the failing test**

```go
// add to internal/tui/health_test.go
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
```

(`strings` is already imported by `hostname_test.go` in this package; if `health_test.go` lacks it, add `"strings"`.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run 'TestNameColor|TestRenderHostItem_OnlineStatus'`
Expected: FAIL — `undefined: nameColor`.

- [ ] **Step 3: Write minimal implementation**

In `internal/tui/health.go`, add:

```go
// nameColor returns the foreground color for a host name given its health, or
// "" when the name should keep its default color (unknown).
func nameColor(status healthStatus) string {
	switch status {
	case healthOnline:
		return colorPositive
	case healthOffline:
		return colorDim
	default:
		return ""
	}
}
```

In `internal/tui/styles.go`, add (this file already imports lipgloss):

```go
// styledName returns the name-cell style for a host, tinted by health status.
func styledName(status healthStatus, width int) lipgloss.Style {
	s := nameStyle.Width(width)
	if c := nameColor(status); c != "" {
		s = s.Foreground(lipgloss.Color(c))
	}
	return s
}
```

In `internal/tui/view.go` `renderHostItem`, replace the name render. Change:

```go
		nameStyle.Width(nameWidth).Render(nameStr),
```

to:

```go
		styledName(m.health[host.Name], nameWidth).Render(nameStr),
```

(`m.health[host.Name]` returns `healthUnknown` for absent keys, so untested/unchecked hosts keep the default color. No signature change — `renderHostItem` is already a `*model` method.)

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/ -run 'TestNameColor|TestRenderHostItem_OnlineStatus' -v`
Expected: PASS

- [ ] **Step 5: Run the full suite + vet**

Run: `go vet ./... && go test ./...`
Expected: all packages PASS (existing `renderHostItem` truncation tests unaffected — unknown status = no color).

- [ ] **Step 6: Commit**

```bash
git add internal/tui/health.go internal/tui/styles.go internal/tui/view.go internal/tui/health_test.go
git commit -m "feat(tui): color host name by health status (green/dim)"
```

---

## Self-Review

**Spec coverage:**
- TCP dial mechanism + timeout → Task 2 (`dialHost`, `healthTimeout`). ✓
- DNS-failure → unknown → Task 1 (`classifyDialError`). ✓
- Background via checkUpdate pattern → Task 2 (`checkHostCmd`). ✓
- Model state map keyed by name → Task 3. ✓
- Startup trigger (batch) + `r` refresh (reset to unknown) → Task 4. ✓
- Name-color display, layout untouched → Task 5. ✓
- Selected-row / search-highlight coexistence → preserved: only the name cell's foreground changes; `highlightMatched` still wraps matched runes (accent) inside, unchanged. ✓
- Non-goals (periodic tick, glyphs, ssh-config parsing, latency, concurrency cap) → not implemented, per spec. ✓

**Placeholder scan:** none — every step has full code and exact commands.

**Type consistency:** `healthStatus`, `healthResultMsg{name,status}`, `checkHostCmd(name,host,port)`, `healthCheckAll(hosts)`, `nameColor(status)`, `styledName(status,width)`, `model.health` used consistently across tasks. `renderHostItem` keeps its existing 5-arg signature (status read from `m.health`).
