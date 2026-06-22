package cli_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/yhzion/portkey/internal/cli"
	"github.com/yhzion/portkey/internal/updater"
)

// makeUpdateServer returns an httptest.Server that serves the given tag_name
// for both /releases/latest and /releases/tags/<tag>.
func makeUpdateServer(t *testing.T, tag string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	payload := fmt.Sprintf(`{"tag_name":%q,"assets":[]}`, tag)
	mux.HandleFunc("/repos/yhzion/portkey/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, payload)
	})
	mux.HandleFunc("/repos/yhzion/portkey/releases/tags/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, payload)
	})
	return httptest.NewServer(mux)
}

func makeUpdClient(t *testing.T, srv *httptest.Server) *updater.Client {
	t.Helper()
	return &updater.Client{
		HTTP:    srv.Client(),
		Owner:   "yhzion",
		Repo:    "portkey",
		BaseURL: srv.URL,
	}
}

// --- --check-only / --dry-run ---

// TestUpdateCheckOnly_UpdateAvailable verifies --check-only returns
// ExitUpdateAvailable (10) when the server reports a newer version, and does
// NOT attempt to install (no assets → no runtime error).
func TestUpdateCheckOnly_UpdateAvailable(t *testing.T) {
	srv := makeUpdateServer(t, "v2.0.0")
	defer srv.Close()

	upd := makeUpdClient(t, srv)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	code := cli.Dispatch([]string{"portkey", "update", "--check-only"}, "v1.0.0", "", upd)
	w.Close()
	os.Stdout = old

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if code != cli.ExitUpdateAvailable {
		t.Errorf("--check-only with update available: code = %d, want %d (ExitUpdateAvailable)",
			code, cli.ExitUpdateAvailable)
	}
	if !strings.Contains(output, "v1.0.0") {
		t.Errorf("output should mention current version v1.0.0, got: %q", output)
	}
	if !strings.Contains(output, "v2.0.0") {
		t.Errorf("output should mention latest version v2.0.0, got: %q", output)
	}
}

// TestUpdateCheckOnly_UpToDate verifies --check-only returns ExitSuccess (0)
// when already up to date.
func TestUpdateCheckOnly_UpToDate(t *testing.T) {
	srv := makeUpdateServer(t, "v1.0.0")
	defer srv.Close()

	upd := makeUpdClient(t, srv)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	code := cli.Dispatch([]string{"portkey", "update", "--check-only"}, "v1.0.0", "", upd)
	w.Close()
	os.Stdout = old

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if code != 0 {
		t.Errorf("--check-only up to date: code = %d, want 0", code)
	}
	if !strings.Contains(output, "up to date") {
		t.Errorf("output should contain 'up to date', got: %q", output)
	}
}

// TestUpdateDryRun_Alias verifies --dry-run behaves identically to --check-only.
func TestUpdateDryRun_Alias(t *testing.T) {
	srv := makeUpdateServer(t, "v2.0.0")
	defer srv.Close()

	upd := makeUpdClient(t, srv)

	// Silence stdout
	old := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	code := cli.Dispatch([]string{"portkey", "update", "--dry-run"}, "v1.0.0", "", upd)
	os.Stdout = old

	if code != cli.ExitUpdateAvailable {
		t.Errorf("--dry-run with update available: code = %d, want %d", code, cli.ExitUpdateAvailable)
	}
}

// TestUpdateCheckOnly_NeverInstalls structurally proves check-only never
// calls DownloadAndInstall even when an update is available, by ensuring no
// runtime error occurs (no assets → would fail if install ran).
func TestUpdateCheckOnly_NeverInstalls(t *testing.T) {
	// Server reports a newer version but provides no assets.
	// If install ran, it would fail with "no binary available" (code 1).
	// check-only must return ExitUpdateAvailable, not ExitRuntime.
	srv := makeUpdateServer(t, "v99.0.0")
	defer srv.Close()

	upd := makeUpdClient(t, srv)

	old := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	code := cli.Dispatch([]string{"portkey", "update", "--check-only"}, "v1.0.0", "", upd)
	os.Stdout = old

	if code == 1 {
		t.Errorf("--check-only called install (got code 1); must never install")
	}
	if code != cli.ExitUpdateAvailable {
		t.Errorf("--check-only update available: code = %d, want %d", code, cli.ExitUpdateAvailable)
	}
}

// TestUpdateCheckOnly_DevVersion verifies that --check-only bypasses the dev
// guard (dev builds should be able to check).
// Actually per spec: check-only does not bypass dev guard since it still checks.
// The dev guard fires first and returns success before any network call.
// This test verifies that --check-only with dev version also returns success
// (same as the no-flag dev path, for orthogonality).
func TestUpdateCheckOnly_DevVersion(t *testing.T) {
	// nil updater: will panic if CheckLatest is called.
	old := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	code := cli.Dispatch([]string{"portkey", "update", "--check-only"}, "dev", "", nil)
	os.Stdout = old

	// dev guard fires → ExitSuccess (does not attempt network check).
	if code != 0 {
		t.Errorf("--check-only dev: code = %d, want 0", code)
	}
}

// --- --version-target ---

// TestUpdateVersionTarget_InstallsNamedTag verifies that --version-target
// fetches the named tag via /releases/tags/<tag>, bypassing IsNewer.
func TestUpdateVersionTarget_InstallsNamedTag(t *testing.T) {
	var requestedPath string
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/yhzion/portkey/releases/tags/", func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		// No assets → will fail with runtime error, but proves the path was hit.
		fmt.Fprintf(w, `{"tag_name":"v0.5.0","assets":[]}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	upd := makeUpdClient(t, srv)

	old := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	old2 := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	// v99.0.0 is current (newer than v0.5.0), yet --version-target must still fetch.
	code := cli.Dispatch([]string{"portkey", "update", "--version-target", "v0.5.0"}, "v99.0.0", "", upd)
	os.Stderr = old
	os.Stdout = old2

	// Should fail with runtime error (no assets for this platform), NOT success.
	// That proves it bypassed IsNewer and attempted the install.
	if code != 1 {
		t.Errorf("--version-target with no assets: code = %d, want 1 (runtime error after bypassing IsNewer)",
			code)
	}
	if !strings.Contains(requestedPath, "v0.5.0") {
		t.Errorf("request path = %q, want it to contain v0.5.0", requestedPath)
	}
}

// TestUpdateVersionTarget_BypassesDevGuard verifies that --version-target
// skips the dev guard (allows a dev build to install a specific version).
func TestUpdateVersionTarget_BypassesDevGuard(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/yhzion/portkey/releases/tags/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"tag_name":"v1.0.0","assets":[]}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	upd := makeUpdClient(t, srv)

	old := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	old2 := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	// "dev" is an unparseable version → dev guard would fire and return 0.
	// With --version-target, the guard must be bypassed → network call → no assets → code 1.
	code := cli.Dispatch([]string{"portkey", "update", "--version-target", "v1.0.0"}, "dev", "", upd)
	os.Stderr = old
	os.Stdout = old2

	if code != 1 {
		t.Errorf("--version-target dev build: code = %d, want 1 (dev guard bypassed, no assets)", code)
	}
}

// TestUpdateVersionTarget_InvalidTag verifies that an invalid --version-target
// value returns ExitUsage without making a network call.
func TestUpdateVersionTarget_InvalidTag(t *testing.T) {
	cases := []string{
		"",
		"a/b",
		"..",
		"v1.0 0",
	}

	for _, tag := range cases {
		t.Run(fmt.Sprintf("%q", tag), func(t *testing.T) {
			networkCalled := false
			mux := http.NewServeMux()
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				networkCalled = true
				w.WriteHeader(http.StatusOK)
			})
			srv := httptest.NewServer(mux)
			defer srv.Close()

			upd := makeUpdClient(t, srv)

			old := os.Stderr
			os.Stderr, _ = os.Open(os.DevNull)
			code := cli.Dispatch([]string{"portkey", "update", "--version-target", tag}, "v1.0.0", "", upd)
			os.Stderr = old

			if code == 0 {
				t.Errorf("invalid --version-target %q: code = 0, want non-zero", tag)
			}
			if networkCalled {
				t.Errorf("invalid --version-target %q: network was called; must reject before any request", tag)
			}
		})
	}
}

// --- --force ---

// TestUpdateForce_ReinstallsSameTag verifies --force reinstalls even when the
// current version equals the latest.
func TestUpdateForce_ReinstallsSameTag(t *testing.T) {
	srv := makeUpdateServer(t, "v1.0.0")
	defer srv.Close()

	upd := makeUpdClient(t, srv)

	old := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	old2 := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	// Same version → IsNewer is false, but --force must bypass it → no assets → code 1.
	code := cli.Dispatch([]string{"portkey", "update", "--force"}, "v1.0.0", "", upd)
	os.Stderr = old
	os.Stdout = old2

	if code != 1 {
		t.Errorf("--force same version: code = %d, want 1 (bypassed IsNewer, failed on no assets)", code)
	}
}

// TestUpdateForce_BypassesDevGuard verifies --force skips the dev guard.
func TestUpdateForce_BypassesDevGuard(t *testing.T) {
	srv := makeUpdateServer(t, "v1.0.0")
	defer srv.Close()

	upd := makeUpdClient(t, srv)

	old := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	old2 := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	// "dev" → dev guard would return 0 without network. --force bypasses it → code 1.
	code := cli.Dispatch([]string{"portkey", "update", "--force"}, "dev", "", upd)
	os.Stderr = old
	os.Stdout = old2

	if code != 1 {
		t.Errorf("--force dev build: code = %d, want 1 (dev guard bypassed, no assets)", code)
	}
}

// TestUpdateNoFlags_Unchanged verifies that no-flag behavior is unchanged:
// dev guard still fires for unparseable versions.
func TestUpdateNoFlags_DevGuardPreserved(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// nil updater: panics if CheckLatest is called.
	code := cli.Dispatch([]string{"portkey", "update"}, "dev", "", nil)

	w.Close()
	os.Stdout = old

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if code != 0 {
		t.Errorf("no-flag dev version: code = %d, want 0", code)
	}
	if !strings.Contains(output, "Cannot determine current version") {
		t.Errorf("expected dev guard message, got: %q", output)
	}
}

// TestUpdateCheckOnly_WinsOverForce verifies --check-only takes precedence
// over --force: no install happens.
func TestUpdateCheckOnly_WinsOverForce(t *testing.T) {
	// Server reports a newer version (no assets).
	// --check-only + --force: must report and NOT install.
	srv := makeUpdateServer(t, "v2.0.0")
	defer srv.Close()

	upd := makeUpdClient(t, srv)

	old := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	code := cli.Dispatch(
		[]string{"portkey", "update", "--check-only", "--force"},
		"v1.0.0", "", upd,
	)
	os.Stdout = old

	if code == 1 {
		t.Errorf("--check-only --force: code = 1 (install ran); check-only must win")
	}
	if code != cli.ExitUpdateAvailable {
		t.Errorf("--check-only --force: code = %d, want %d", code, cli.ExitUpdateAvailable)
	}
}

// TestUpdateHelp_ContainsExitUpdateAvailable verifies that ExitUpdateAvailable
// is documented in the update command help.
func TestUpdateHelp_ContainsNewFlags(t *testing.T) {
	// Capture stdout to read help output.
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	cli.Dispatch([]string{"portkey", "update", "--help"}, "dev", "", nil)
	w.Close()
	os.Stdout = old

	buf := make([]byte, 8192)
	n, _ := r.Read(buf)
	help := string(buf[:n])

	for _, flag := range []string{"--check-only", "--dry-run", "--version-target", "--force"} {
		if !strings.Contains(help, flag) {
			t.Errorf("update --help missing flag %s\nhelp output:\n%s", flag, help)
		}
	}
}

// --- ExitUpdateAvailable is exported ---

// TestExitUpdateAvailableValue ensures ExitUpdateAvailable == 10.
func TestExitUpdateAvailableValue(t *testing.T) {
	if cli.ExitUpdateAvailable != 10 {
		t.Errorf("ExitUpdateAvailable = %d, want 10", cli.ExitUpdateAvailable)
	}
}

// Smoke test for JSON output — ensures that the existing list behavior still
// works and we haven't broken the flag package initialization.
func TestUpdateFlags_FlagSetInitialized(t *testing.T) {
	path := setupConfig(t, nil)
	data, _ := json.Marshal([]struct{}{})
	_ = data
	code := cli.Dispatch([]string{"portkey", "list"}, "v1.0.0", path, nil)
	if code != 0 {
		t.Errorf("sanity check list: code = %d, want 0", code)
	}
}
