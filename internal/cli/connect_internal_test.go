package cli

import (
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
