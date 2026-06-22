package cli_test

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yhzion/portkey/internal/cli"
	"github.com/yhzion/portkey/internal/config"
	"github.com/yhzion/portkey/internal/updater"
)

// helper creates a temp config file with given hosts and returns the path.
func setupConfig(t *testing.T, hosts []config.Host) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "hosts.json")
	cfg := &config.Config{Hosts: hosts}
	store := config.NewStore(path)
	if err := store.Save(cfg); err != nil {
		t.Fatal(err)
	}
	return path
}

func configFromPath(t *testing.T, path string) *config.Config {
	t.Helper()
	cfg, err := config.NewStore(path).Load()
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

// --- Dispatch: unknown subcommand ---

func TestDispatchUnknownSubcommand(t *testing.T) {
	path := setupConfig(t, nil)
	code := cli.Dispatch([]string{"portkey", "bogus"}, "dev", path, nil)
	if code != 2 {
		t.Errorf("Dispatch(bogus) = %d, want 2", code)
	}
}

// --- list ---

func TestDispatchListDefault(t *testing.T) {
	path := setupConfig(t, []config.Host{
		{Name: "prod", Username: "admin", Host: "10.0.0.1", Port: 22},
	})
	code := cli.Dispatch([]string{"portkey", "list"}, "dev", path, nil)
	if code != 0 {
		t.Errorf("list = %d, want 0", code)
	}
}

func TestDispatchListEmpty(t *testing.T) {
	path := setupConfig(t, nil)
	code := cli.Dispatch([]string{"portkey", "list"}, "dev", path, nil)
	if code != 0 {
		t.Errorf("list (empty) = %d, want 0", code)
	}
}

func TestDispatchListJSON(t *testing.T) {
	path := setupConfig(t, []config.Host{
		{Name: "prod", Username: "admin", Host: "10.0.0.1", Port: 22},
	})
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	code := cli.Dispatch([]string{"portkey", "list", "--json"}, "dev", path, nil)
	w.Close()
	os.Stdout = old

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if code != 0 {
		t.Errorf("list --json = %d, want 0", code)
	}
	var hosts []config.Host
	if err := json.Unmarshal([]byte(output), &hosts); err != nil {
		t.Fatalf("invalid JSON output: %s\nraw: %q", err, output)
	}
	if len(hosts) != 1 {
		t.Errorf("len(hosts) = %d, want 1", len(hosts))
	}
	if hosts[0].Name != "prod" {
		t.Errorf("hosts[0].Name = %q, want %q", hosts[0].Name, "prod")
	}
}

// --- add ---

func TestDispatchAdd(t *testing.T) {
	path := setupConfig(t, nil)
	code := cli.Dispatch([]string{
		"portkey", "add",
		"--name", "myhost",
		"--user", "admin",
		"--host", "10.0.0.1",
	}, "dev", path, nil)
	if code != 0 {
		t.Fatalf("add = %d, want 0", code)
	}

	cfg := configFromPath(t, path)
	if len(cfg.Hosts) != 1 {
		t.Fatalf("len(Hosts) = %d, want 1", len(cfg.Hosts))
	}
	if cfg.Hosts[0].Name != "myhost" {
		t.Errorf("Name = %q, want %q", cfg.Hosts[0].Name, "myhost")
	}
	if cfg.Hosts[0].Username != "admin" {
		t.Errorf("Username = %q, want %q", cfg.Hosts[0].Username, "admin")
	}
	if cfg.Hosts[0].Host != "10.0.0.1" {
		t.Errorf("Host = %q, want %q", cfg.Hosts[0].Host, "10.0.0.1")
	}
	if cfg.Hosts[0].Port != 22 {
		t.Errorf("Port = %d, want 22 (default)", cfg.Hosts[0].Port)
	}
}

func TestDispatchAddWithPort(t *testing.T) {
	path := setupConfig(t, nil)
	code := cli.Dispatch([]string{
		"portkey", "add",
		"--name", "myhost",
		"--user", "admin",
		"--host", "10.0.0.1",
		"--port", "2222",
	}, "dev", path, nil)
	if code != 0 {
		t.Fatalf("add --port = %d, want 0", code)
	}

	cfg := configFromPath(t, path)
	if cfg.Hosts[0].Port != 2222 {
		t.Errorf("Port = %d, want 2222", cfg.Hosts[0].Port)
	}
}

func TestDispatchAddDuplicateName(t *testing.T) {
	path := setupConfig(t, []config.Host{
		{Name: "myhost", Username: "u", Host: "h", Port: 22},
	})
	// Capture stderr to suppress output during test
	old := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	code := cli.Dispatch([]string{
		"portkey", "add",
		"--name", "myhost",
		"--user", "admin",
		"--host", "10.0.0.2",
	}, "dev", path, nil)
	os.Stderr = old

	if code != 2 {
		t.Errorf("add duplicate = %d, want 2", code)
	}
}

func TestDispatchAddInvalidName(t *testing.T) {
	path := setupConfig(t, nil)
	old := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	code := cli.Dispatch([]string{
		"portkey", "add",
		"--name", "Bad Name!",
		"--user", "admin",
		"--host", "10.0.0.1",
	}, "dev", path, nil)
	os.Stderr = old

	if code != 2 {
		t.Errorf("add invalid name = %d, want 2", code)
	}
}

func TestDispatchAddMissingRequired(t *testing.T) {
	path := setupConfig(t, nil)
	old := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	code := cli.Dispatch([]string{
		"portkey", "add",
		"--name", "myhost",
	}, "dev", path, nil)
	os.Stderr = old

	if code != 2 {
		t.Errorf("add missing fields = %d, want 2", code)
	}
}

func TestDispatchAddPortOutOfRange(t *testing.T) {
	path := setupConfig(t, nil)
	old := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	code := cli.Dispatch([]string{
		"portkey", "add",
		"--name", "myhost",
		"--user", "admin",
		"--host", "10.0.0.1",
		"--port", "99999",
	}, "dev", path, nil)
	os.Stderr = old

	if code != 2 {
		t.Errorf("add bad port = %d, want 2", code)
	}
}

// --- edit ---

func TestDispatchEdit(t *testing.T) {
	path := setupConfig(t, []config.Host{
		{Name: "myhost", Username: "admin", Host: "10.0.0.1", Port: 22},
	})
	code := cli.Dispatch([]string{
		"portkey", "edit",
		"--name", "myhost",
		"--user", "root",
	}, "dev", path, nil)
	if code != 0 {
		t.Fatalf("edit = %d, want 0", code)
	}

	cfg := configFromPath(t, path)
	if cfg.Hosts[0].Username != "root" {
		t.Errorf("Username = %q, want %q", cfg.Hosts[0].Username, "root")
	}
	// Unchanged fields preserved
	if cfg.Hosts[0].Host != "10.0.0.1" {
		t.Errorf("Host changed unexpectedly to %q", cfg.Hosts[0].Host)
	}
}

func TestDispatchEditRename(t *testing.T) {
	path := setupConfig(t, []config.Host{
		{Name: "myhost", Username: "admin", Host: "10.0.0.1", Port: 22},
	})
	code := cli.Dispatch([]string{
		"portkey", "edit",
		"--name", "myhost",
		"--new-name", "renamed",
	}, "dev", path, nil)
	if code != 0 {
		t.Fatalf("edit rename = %d, want 0", code)
	}

	cfg := configFromPath(t, path)
	if cfg.Hosts[0].Name != "renamed" {
		t.Errorf("Name = %q, want %q", cfg.Hosts[0].Name, "renamed")
	}
}

// TestDispatchEditSuffixMatchNonInteractiveAborts verifies that a suffix match
// on edit in a non-interactive context (the default under test, where stdin is
// not a TTY) aborts rather than silently editing the matched host (issue #46).
func TestDispatchEditSuffixMatchNonInteractiveAborts(t *testing.T) {
	path := setupConfig(t, []config.Host{
		{Name: "production-api", Username: "admin", Host: "10.0.0.1", Port: 22},
	})
	old := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	code := cli.Dispatch([]string{
		"portkey", "edit",
		"--name", "api",
		"--user", "root",
	}, "dev", path, nil)
	os.Stderr = old

	if code != 1 {
		t.Fatalf("edit suffix in non-interactive context = %d, want 1 (abort)", code)
	}

	// The host must be left untouched.
	cfg := configFromPath(t, path)
	if cfg.Hosts[0].Username != "admin" {
		t.Errorf("host was modified after non-interactive suffix abort: Username = %q, want admin",
			cfg.Hosts[0].Username)
	}
}

func TestDispatchEditNotFound(t *testing.T) {
	path := setupConfig(t, nil)
	old := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	code := cli.Dispatch([]string{
		"portkey", "edit",
		"--name", "nope",
		"--user", "root",
	}, "dev", path, nil)
	os.Stderr = old

	if code != 1 {
		t.Errorf("edit not found = %d, want 1", code)
	}
}

func TestDispatchEditAmbiguous(t *testing.T) {
	path := setupConfig(t, []config.Host{
		{Name: "prod-api", Username: "u", Host: "h1", Port: 22},
		{Name: "staging-api", Username: "u", Host: "h2", Port: 22},
	})
	old := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	code := cli.Dispatch([]string{
		"portkey", "edit",
		"--name", "api",
		"--user", "root",
	}, "dev", path, nil)
	os.Stderr = old

	if code != 1 {
		t.Errorf("edit ambiguous = %d, want 1", code)
	}
}

// --- delete ---

func TestDispatchDelete(t *testing.T) {
	path := setupConfig(t, []config.Host{
		{Name: "myhost", Username: "admin", Host: "10.0.0.1", Port: 22},
	})
	// Capture stdin to provide "y" confirmation
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	w.Write([]byte("y\n"))
	w.Close()
	os.Stdin = r

	code := cli.Dispatch([]string{
		"portkey", "delete",
		"--name", "myhost",
	}, "dev", path, nil)

	r.Close()
	os.Stdin = oldStdin

	if code != 0 {
		t.Fatalf("delete = %d, want 0", code)
	}

	cfg := configFromPath(t, path)
	if len(cfg.Hosts) != 0 {
		t.Errorf("len(Hosts) = %d, want 0", len(cfg.Hosts))
	}
}

func TestDispatchDeleteForce(t *testing.T) {
	path := setupConfig(t, []config.Host{
		{Name: "myhost", Username: "admin", Host: "10.0.0.1", Port: 22},
	})
	code := cli.Dispatch([]string{
		"portkey", "delete",
		"--name", "myhost",
		"--force",
	}, "dev", path, nil)
	if code != 0 {
		t.Fatalf("delete --force = %d, want 0", code)
	}

	cfg := configFromPath(t, path)
	if len(cfg.Hosts) != 0 {
		t.Errorf("len(Hosts) = %d, want 0", len(cfg.Hosts))
	}
}

func TestDispatchDeleteNotFound(t *testing.T) {
	path := setupConfig(t, nil)
	old := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	code := cli.Dispatch([]string{
		"portkey", "delete",
		"--name", "nope",
		"--force",
	}, "dev", path, nil)
	os.Stderr = old

	if code != 1 {
		t.Errorf("delete not found = %d, want 1", code)
	}
}

// --- connect ---

func TestDispatchConnectNotFound(t *testing.T) {
	path := setupConfig(t, nil)
	old := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	code := cli.Dispatch([]string{
		"portkey", "connect",
		"--name", "nope",
	}, "dev", path, nil)
	os.Stderr = old

	if code != 1 {
		t.Errorf("connect not found = %d, want 1", code)
	}
}

func TestDispatchConnectInvalidPort(t *testing.T) {
	path := setupConfig(t, []config.Host{
		{Name: "myhost", Username: "admin", Host: "10.0.0.1", Port: 22},
	})
	old := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	code := cli.Dispatch([]string{
		"portkey", "connect",
		"--name", "myhost",
		"--port", "99999",
	}, "dev", path, nil)
	os.Stderr = old

	if code != 2 {
		t.Errorf("connect bad port = %d, want 2", code)
	}
}

// --- help ---

func TestDispatchHelpList(t *testing.T) {
	code := cli.Dispatch([]string{"portkey", "list", "--help"}, "dev", "", nil)
	if code != 0 {
		t.Errorf("list --help = %d, want 0", code)
	}
}

func TestDispatchHelpAdd(t *testing.T) {
	code := cli.Dispatch([]string{"portkey", "add", "--help"}, "dev", "", nil)
	if code != 0 {
		t.Errorf("add --help = %d, want 0", code)
	}
}

// --- version ---

func TestDispatchVersion(t *testing.T) {
	code := cli.Dispatch([]string{"portkey", "--version"}, "1.2.3", "", nil)
	if code != 0 {
		t.Errorf("--version = %d, want 0", code)
	}
}

func TestDispatchHelp(t *testing.T) {
	code := cli.Dispatch([]string{"portkey", "--help"}, "1.2.3", "", nil)
	if code != 0 {
		t.Errorf("--help = %d, want 0", code)
	}
}

// --- config file not found (graceful for list) ---

func TestDispatchListNoConfig(t *testing.T) {
	code := cli.Dispatch([]string{"portkey", "list"}, "dev", "/nonexistent/hosts.json", nil)
	if code != 0 {
		t.Errorf("list no config = %d, want 0 (empty list)", code)
	}
}

// --- Verify list output contains host data ---

func TestDispatchListContainsHostData(t *testing.T) {
	path := setupConfig(t, []config.Host{
		{Name: "prod", Username: "admin", Host: "10.0.0.1", Port: 22},
	})
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	code := cli.Dispatch([]string{"portkey", "list"}, "dev", path, nil)
	w.Close()
	os.Stdout = old

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if code != 0 {
		t.Errorf("list = %d, want 0", code)
	}
	if !strings.Contains(output, "prod") {
		t.Errorf("list output should contain host name, got: %q", output)
	}
	if !strings.Contains(output, "admin") {
		t.Errorf("list output should contain username, got: %q", output)
	}
}

// --- update ---

func TestDispatchUpdateHelp(t *testing.T) {
	code := cli.Dispatch([]string{"portkey", "update", "--help"}, "dev", "", nil)
	if code != 0 {
		t.Errorf("update --help = %d, want 0", code)
	}
}

func TestDispatchUpdateAlreadyUpToDate(t *testing.T) {
	// Use a mock server to avoid real GitHub API calls (flaky on CI due to rate limiting).
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/yhzion/portkey/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"tag_name": "v0.0.1",
			"assets": []
		}`)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	upd := &updater.Client{
		HTTP:    server.Client(),
		Owner:   "yhzion",
		Repo:    "portkey",
		BaseURL: server.URL,
	}

	code := cli.Dispatch([]string{"portkey", "update"}, "v99.0.0", "", upd)
	if code != 0 {
		t.Errorf("update up-to-date = %d, want 0", code)
	}
}

func TestDispatchUpdateNewerAvailable(t *testing.T) {
	// Use a mock server to simulate GitHub API
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/yhzion/portkey/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"tag_name": "v0.2.0",
			"assets": []
		}`)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	upd := &updater.Client{
		HTTP:    server.Client(),
		Owner:   "yhzion",
		Repo:    "portkey",
		BaseURL: server.URL,
	}

	// Capture stderr for error message
	old := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	code := cli.Dispatch([]string{"portkey", "update"}, "v0.1.0", "", upd)
	os.Stderr = old

	// Will fail because no assets, but should get past the version check
	if code != 1 {
		t.Errorf("update with no assets = %d, want 1 (runtime error)", code)
	}
}

// TestDispatchUpdateDevVersionSkips verifies that running `portkey update` with
// an unparseable version (e.g. "dev") prints a "Cannot determine current
// version" message, returns ExitSuccess, and makes no network call at all.
// The nil updater ensures any accidental network call panics, proving the guard
// short-circuits before CheckLatest is ever invoked.
func TestDispatchUpdateDevVersionSkips(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// nil updater: if CheckLatest is called it will panic (nil pointer dereference),
	// which would fail the test loudly — proving the guard fires first.
	code := cli.Dispatch([]string{"portkey", "update"}, "dev", "", nil)

	w.Close()
	os.Stdout = old

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if code != 0 {
		t.Errorf("update dev version = %d, want 0 (ExitSuccess)", code)
	}
	if !strings.Contains(output, "Cannot determine current version") {
		t.Errorf("expected 'Cannot determine current version' in output, got: %q", output)
	}
	if strings.Contains(output, "Already up to date") {
		t.Errorf("output should NOT contain 'Already up to date' for dev version, got: %q", output)
	}
}

// TestDispatchUpdateUnparseableVersionSkips checks that any non-semver version
// string (not just "dev") also triggers the guard.
func TestDispatchUpdateUnparseableVersionSkips(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	code := cli.Dispatch([]string{"portkey", "update"}, "dirty-branch", "", nil)

	w.Close()
	os.Stdout = old

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if code != 0 {
		t.Errorf("update unparseable version = %d, want 0 (ExitSuccess)", code)
	}
	if !strings.Contains(output, "Cannot determine current version") {
		t.Errorf("expected 'Cannot determine current version' in output, got: %q", output)
	}
}

// --- help-flag synchronization ---

func TestCommandHelpContainsAllFlags(t *testing.T) {
	for _, cmd := range cli.Commands {
		if cmd.Flags == nil {
			continue
		}
		fs := flag.NewFlagSet(cmd.Name, flag.ContinueOnError)
		cmd.Flags(fs)
		help := cmd.Help()

		fs.VisitAll(func(f *flag.Flag) {
			if !strings.Contains(help, "--"+f.Name) {
				t.Errorf("%s: help missing flag --%s", cmd.Name, f.Name)
			}
		})
	}
}
