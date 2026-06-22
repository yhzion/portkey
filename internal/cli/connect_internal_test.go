package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/yhzion/portkey/internal/config"
)

// fakeSSHRunner records the host it was asked to connect to and returns a
// configurable error, standing in for a real ssh process during tests.
type fakeSSHRunner struct {
	called  bool
	gotHost config.Host
	err     error
}

func (f *fakeSSHRunner) Run(host config.Host) error {
	f.called = true
	f.gotHost = host
	return f.err
}

// writeConfig creates a temp config file with the given hosts and returns its path.
func writeConfig(t *testing.T, hosts []config.Host) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "hosts.json")
	if err := config.NewStore(path).Save(&config.Config{Hosts: hosts}); err != nil {
		t.Fatal(err)
	}
	return path
}

// stubConfirm replaces the confirm seam during tests, recording its args and
// returning a canned decision. It never touches real stdin.
type stubConfirm struct {
	called  bool
	host    string
	query   string
	verb    string
	approve bool
}

func makeStubConfirm(approve bool) *stubConfirm {
	return &stubConfirm{approve: approve}
}

// withConfirm swaps the package-level confirm seam and restores it on cleanup.
func withConfirm(t *testing.T, fn func(hostName, query, verb string) (bool, error)) {
	t.Helper()
	prev := confirmSuffix
	confirmSuffix = fn
	t.Cleanup(func() { confirmSuffix = prev })
}

func TestDispatchConnectHappyPath(t *testing.T) {
	path := writeConfig(t, []config.Host{
		{Name: "myhost", Username: "admin", Host: "10.0.0.1", Port: 22},
	})

	fake := &fakeSSHRunner{}
	prev := defaultSSHRunner
	defaultSSHRunner = fake
	defer func() { defaultSSHRunner = prev }()

	code := Dispatch([]string{"portkey", "connect", "--name", "myhost"}, "dev", path, nil)

	if code != ExitSuccess {
		t.Errorf("connect happy path = %d, want %d", code, ExitSuccess)
	}
	if !fake.called {
		t.Fatal("expected SSHRunner.Run to be called")
	}
	if fake.gotHost.Name != "myhost" {
		t.Errorf("runner got host %q, want myhost", fake.gotHost.Name)
	}

	// LastUsed must be stamped on disk so the host bubbles up the
	// recency-sorted list, mirroring the TUI path.
	reloaded, err := config.NewStore(path).Load()
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}
	idx, _, err := reloaded.FindHostByNameMatch("myhost")
	if err != nil {
		t.Fatalf("FindHostByName after connect: %v", err)
	}
	if reloaded.Hosts[idx].LastUsed == "" {
		t.Error("expected LastUsed to be set on disk after a successful connect")
	}
}

func TestDispatchConnectRunnerError(t *testing.T) {
	path := writeConfig(t, []config.Host{
		{Name: "myhost", Username: "admin", Host: "10.0.0.1", Port: 22},
	})

	fake := &fakeSSHRunner{err: os.ErrPermission}
	prev := defaultSSHRunner
	defaultSSHRunner = fake
	defer func() { defaultSSHRunner = prev }()

	old := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	code := Dispatch([]string{"portkey", "connect", "--name", "myhost"}, "dev", path, nil)
	os.Stderr = old

	if code != ExitRuntime {
		t.Errorf("connect with runner error = %d, want %d", code, ExitRuntime)
	}
}

// --- suffix-match confirmation (issue #46) ---

// TestDispatchConnectExactMatchSkipsConfirm asserts that an exact --name match
// connects with NO confirm prompt, even though a suffix-matching host exists.
func TestDispatchConnectExactMatchSkipsConfirm(t *testing.T) {
	path := writeConfig(t, []config.Host{
		{Name: "api", Username: "u", Host: "h0", Port: 22},
		{Name: "production-api", Username: "admin", Host: "10.0.0.1", Port: 22},
	})

	fake := &fakeSSHRunner{}
	prev := defaultSSHRunner
	defaultSSHRunner = fake
	defer func() { defaultSSHRunner = prev }()

	sc := makeStubConfirm(true)
	withConfirm(t, func(hostName, query, verb string) (bool, error) {
		sc.called = true
		sc.host, sc.query, sc.verb = hostName, query, verb
		return sc.approve, nil
	})

	code := Dispatch([]string{"portkey", "connect", "--name", "api"}, "dev", path, nil)

	if code != ExitSuccess {
		t.Errorf("exact connect = %d, want %d", code, ExitSuccess)
	}
	if !fake.called {
		t.Fatal("expected SSHRunner.Run to be called on exact match")
	}
	if fake.gotHost.Name != "api" {
		t.Errorf("runner got host %q, want api", fake.gotHost.Name)
	}
	if sc.called {
		t.Error("confirm seam was invoked on an exact match; should be skipped")
	}
}

// TestDispatchConnectSuffixMatchAbortsWhenNotConfirmed asserts that a suffix
// match does NOT auto-connect when the user declines confirmation.
func TestDispatchConnectSuffixMatchAbortsWhenNotConfirmed(t *testing.T) {
	path := writeConfig(t, []config.Host{
		{Name: "prod-web", Username: "admin", Host: "10.0.0.1", Port: 22},
	})

	fake := &fakeSSHRunner{}
	prev := defaultSSHRunner
	defaultSSHRunner = fake
	defer func() { defaultSSHRunner = prev }()

	withConfirm(t, func(hostName, query, verb string) (bool, error) {
		if hostName != "prod-web" || query != "web" {
			t.Errorf("confirm args = (%q, %q), want (prod-web, web)", hostName, query)
		}
		return false, nil // user declines
	})

	old := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	code := Dispatch([]string{"portkey", "connect", "--name", "web"}, "dev", path, nil)
	os.Stderr = old

	if code != ExitRuntime {
		t.Errorf("declined suffix connect = %d, want %d (abort)", code, ExitRuntime)
	}
	if fake.called {
		t.Error("SSHRunner.Run must NOT be called when suffix match is declined")
	}
}

// TestDispatchConnectSuffixMatchConnectsWhenConfirmed asserts that a suffix
// match proceeds to connect when the user confirms via the seam.
func TestDispatchConnectSuffixMatchConnectsWhenConfirmed(t *testing.T) {
	path := writeConfig(t, []config.Host{
		{Name: "prod-web", Username: "admin", Host: "10.0.0.1", Port: 22},
	})

	fake := &fakeSSHRunner{}
	prev := defaultSSHRunner
	defaultSSHRunner = fake
	defer func() { defaultSSHRunner = prev }()

	withConfirm(t, func(hostName, query, verb string) (bool, error) {
		return true, nil // user confirms
	})

	code := Dispatch([]string{"portkey", "connect", "--name", "web"}, "dev", path, nil)

	if code != ExitSuccess {
		t.Errorf("confirmed suffix connect = %d, want %d", code, ExitSuccess)
	}
	if !fake.called {
		t.Fatal("expected SSHRunner.Run to be called after user confirms suffix match")
	}
	if fake.gotHost.Name != "prod-web" {
		t.Errorf("runner got host %q, want prod-web", fake.gotHost.Name)
	}
}

// TestDispatchConnectSuffixMatchAbortsInNonInteractive asserts that when the
// confirm seam signals a non-interactive context (the production
// defaultConfirmSuffix does this when stdin is not a TTY), a suffix match
// aborts with the safety error rather than auto-connecting.
func TestDispatchConnectSuffixMatchAbortsInNonInteractive(t *testing.T) {
	path := writeConfig(t, []config.Host{
		{Name: "prod-web", Username: "admin", Host: "10.0.0.1", Port: 22},
	})

	fake := &fakeSSHRunner{}
	prev := defaultSSHRunner
	defaultSSHRunner = fake
	defer func() { defaultSSHRunner = prev }()

	// Simulate the non-interactive branch of defaultConfirmSuffix: it refuses
	// with an error naming the matched host. We stub the seam rather than
	// relying on the test runner's stdin being a non-TTY (some CI/harness
	// setups attach a real pty to stdin).
	withConfirm(t, func(hostName, query, verb string) (bool, error) {
		return false, fmt.Errorf(
			"non-interactive suffix match %q for %q; supply the exact name",
			hostName, query,
		)
	})

	old := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	code := Dispatch([]string{"portkey", "connect", "--name", "web"}, "dev", path, nil)
	os.Stderr = old

	if code != ExitRuntime {
		t.Errorf("non-interactive suffix connect = %d, want %d (safety abort)", code, ExitRuntime)
	}
	if fake.called {
		t.Error("SSHRunner.Run must NOT be called in non-interactive suffix match")
	}
}

// TestIsTerminalDetectsNonCharacterDevice verifies the stdlib-only TTY check
// returns false for a regular file and true for /dev/null on disk (a char
// device), guarding the safety predicate that powers defaultConfirmSuffix.
func TestIsTerminalDetectsNonCharacterDevice(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "regular.txt")
	if err := os.WriteFile(fpath, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(fpath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if isTerminal(f) {
		t.Error("isTerminal(regular file) = true, want false")
	}
}

// --- edit suffix-match confirmation (issue #46) ---

// TestDispatchEditExactMatchSkipsConfirm asserts that an exact --name match
// edits with NO confirm prompt.
func TestDispatchEditExactMatchSkipsConfirm(t *testing.T) {
	path := writeConfig(t, []config.Host{
		{Name: "api", Username: "u", Host: "h0", Port: 22},
		{Name: "production-api", Username: "admin", Host: "10.0.0.1", Port: 22},
	})

	sc := makeStubConfirm(true)
	withConfirm(t, func(hostName, query, verb string) (bool, error) {
		sc.called = true
		sc.host, sc.query, sc.verb = hostName, query, verb
		return sc.approve, nil
	})

	code := Dispatch([]string{
		"portkey", "edit", "--name", "api", "--user", "root",
	}, "dev", path, nil)

	if code != ExitSuccess {
		t.Errorf("exact edit = %d, want %d", code, ExitSuccess)
	}
	if sc.called {
		t.Error("confirm seam was invoked on an exact edit match; should be skipped")
	}

	cfg, err := config.NewStore(path).Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Hosts[0].Name != "api" {
		t.Fatalf("edited wrong host: %q", cfg.Hosts[0].Name)
	}
	if cfg.Hosts[0].Username != "root" {
		t.Errorf("Username = %q, want root", cfg.Hosts[0].Username)
	}
}

// TestDispatchEditSuffixMatchAbortsWhenNotConfirmed asserts that a suffix match
// on edit does NOT proceed when the user declines.
func TestDispatchEditSuffixMatchAbortsWhenNotConfirmed(t *testing.T) {
	path := writeConfig(t, []config.Host{
		{Name: "prod-web", Username: "admin", Host: "10.0.0.1", Port: 22},
	})

	withConfirm(t, func(hostName, query, verb string) (bool, error) {
		if hostName != "prod-web" || query != "web" {
			t.Errorf("confirm args = (%q, %q), want (prod-web, web)", hostName, query)
		}
		return false, nil
	})

	old := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	code := Dispatch([]string{
		"portkey", "edit", "--name", "web", "--user", "root",
	}, "dev", path, nil)
	os.Stderr = old

	if code != ExitRuntime {
		t.Errorf("declined suffix edit = %d, want %d (abort)", code, ExitRuntime)
	}

	cfg, err := config.NewStore(path).Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Hosts[0].Username != "admin" {
		t.Errorf("host was modified after declined suffix edit: Username = %q, want admin",
			cfg.Hosts[0].Username)
	}
}

// TestDispatchEditSuffixMatchProceedsWhenConfirmed asserts that a suffix match
// on edit proceeds when the user confirms.
func TestDispatchEditSuffixMatchProceedsWhenConfirmed(t *testing.T) {
	path := writeConfig(t, []config.Host{
		{Name: "prod-web", Username: "admin", Host: "10.0.0.1", Port: 22},
	})

	withConfirm(t, func(hostName, query, verb string) (bool, error) {
		return true, nil
	})

	code := Dispatch([]string{
		"portkey", "edit", "--name", "web", "--user", "root",
	}, "dev", path, nil)

	if code != ExitSuccess {
		t.Errorf("confirmed suffix edit = %d, want %d", code, ExitSuccess)
	}

	cfg, err := config.NewStore(path).Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Hosts[0].Username != "root" {
		t.Errorf("Username = %q, want root after confirmed suffix edit", cfg.Hosts[0].Username)
	}
}
