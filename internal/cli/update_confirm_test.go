package cli

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/yhzion/portkey/internal/updater"
)

// makeUpdateSrvNoAssets returns a server that serves a given tag with no
// assets, so any real DownloadAndInstall call fails with ExitRuntime (1).
func makeUpdateSrvNoAssets(t *testing.T, tag string) *httptest.Server {
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

func makeUpdClientInternal(t *testing.T, srv *httptest.Server) *updater.Client {
	t.Helper()
	return &updater.Client{
		HTTP:    srv.Client(),
		Owner:   "yhzion",
		Repo:    "portkey",
		BaseURL: srv.URL,
	}
}

// withConfirmUpdate swaps the package-level confirmUpdate seam and restores it
// on cleanup. Must run in an internal test (package cli) to access the seam.
func withConfirmUpdate(t *testing.T, fn func(current, tag string, force, versionTargetSet bool) (bool, error)) {
	t.Helper()
	prev := confirmUpdate
	confirmUpdate = fn
	t.Cleanup(func() { confirmUpdate = prev })
}

// confirmCalled is a helper that returns a stub recording whether it was invoked.
type confirmRecord struct {
	called bool
}

// --- --yes / -y flag: skips confirm ---

// TestUpdateYesFlag_SkipsConfirm verifies that --yes prevents calling confirmUpdate.
func TestUpdateYesFlag_SkipsConfirm(t *testing.T) {
	srv := makeUpdateSrvNoAssets(t, "v2.0.0")
	defer srv.Close()
	upd := makeUpdClientInternal(t, srv)

	rec := &confirmRecord{}
	withConfirmUpdate(t, func(current, tag string, force, versionTargetSet bool) (bool, error) {
		rec.called = true
		return true, nil
	})

	old := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	old2 := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	// newer available → install path; no assets → code 1 (confirms install was attempted)
	code := Dispatch([]string{"portkey", "update", "--yes"}, "v1.0.0", "", upd)
	os.Stderr = old
	os.Stdout = old2

	if rec.called {
		t.Error("confirmUpdate was called with --yes; it must be skipped")
	}
	// No assets → ExitRuntime (1) proves install was attempted (gate did not block)
	if code != ExitRuntime {
		t.Errorf("--yes newer version: code = %d, want %d (ExitRuntime after skipped confirm)", code, ExitRuntime)
	}
}

// TestUpdateYShortFlag_SkipsConfirm verifies that -y also skips the confirm.
func TestUpdateYShortFlag_SkipsConfirm(t *testing.T) {
	srv := makeUpdateSrvNoAssets(t, "v2.0.0")
	defer srv.Close()
	upd := makeUpdClientInternal(t, srv)

	rec := &confirmRecord{}
	withConfirmUpdate(t, func(current, tag string, force, versionTargetSet bool) (bool, error) {
		rec.called = true
		return true, nil
	})

	old := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	old2 := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	code := Dispatch([]string{"portkey", "update", "-y"}, "v1.0.0", "", upd)
	os.Stderr = old
	os.Stdout = old2

	if rec.called {
		t.Error("confirmUpdate was called with -y; it must be skipped")
	}
	if code != ExitRuntime {
		t.Errorf("-y newer version: code = %d, want %d", code, ExitRuntime)
	}
}

// --- interactive deny → Canceled., ExitSuccess, no install ---

// TestUpdateConfirmDeny_CancelsWithSuccess verifies that a deny from the seam
// prints "Canceled." and returns ExitSuccess WITHOUT installing (prove by
// using a server with no assets: if install ran, code would be ExitRuntime).
func TestUpdateConfirmDeny_CancelsWithSuccess(t *testing.T) {
	srv := makeUpdateSrvNoAssets(t, "v2.0.0")
	defer srv.Close()
	upd := makeUpdClientInternal(t, srv)

	withConfirmUpdate(t, func(current, tag string, force, versionTargetSet bool) (bool, error) {
		return false, nil // user denies
	})

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	code := Dispatch([]string{"portkey", "update"}, "v1.0.0", "", upd)
	w.Close()
	os.Stdout = old

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if code != ExitSuccess {
		t.Errorf("confirm deny: code = %d, want %d (ExitSuccess)", code, ExitSuccess)
	}
	if !strings.Contains(output, "Canceled.") {
		t.Errorf("confirm deny: output = %q, want 'Canceled.' in output", output)
	}
}

// --- interactive confirm → proceeds to install ---

// TestUpdateConfirmApprove_Installs verifies that an approval from the seam
// proceeds to DownloadAndInstall (no assets → ExitRuntime proves it ran).
func TestUpdateConfirmApprove_Installs(t *testing.T) {
	srv := makeUpdateSrvNoAssets(t, "v2.0.0")
	defer srv.Close()
	upd := makeUpdClientInternal(t, srv)

	withConfirmUpdate(t, func(current, tag string, force, versionTargetSet bool) (bool, error) {
		return true, nil // user confirms
	})

	old := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	old2 := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	code := Dispatch([]string{"portkey", "update"}, "v1.0.0", "", upd)
	os.Stderr = old
	os.Stdout = old2

	// No assets → ExitRuntime (1): confirms install was attempted.
	if code != ExitRuntime {
		t.Errorf("confirm approve: code = %d, want %d (ExitRuntime, install attempted)", code, ExitRuntime)
	}
}

// --- non-TTY without --yes → proceeds without prompting ---

// TestDefaultConfirmUpdate_NonTTY_ProceedsWithoutPrompt exercises the REAL
// non-TTY branch of defaultConfirmUpdate directly. It replaces os.Stdin with
// the read end of an os.Pipe (which is never a TTY), so isTerminal returns
// false and defaultConfirmUpdate must return (true, nil) without reading any
// input or printing a prompt.
func TestDefaultConfirmUpdate_NonTTY_ProceedsWithoutPrompt(t *testing.T) {
	// os.Pipe read end is not a terminal.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	defer r.Close()
	defer w.Close()

	origStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = origStdin }()

	// Call the real function — must not block waiting for input.
	ok, cerr := defaultConfirmUpdate("v1.0.0", "v2.0.0", false, false)
	if cerr != nil {
		t.Errorf("defaultConfirmUpdate non-TTY: unexpected error: %v", cerr)
	}
	if !ok {
		t.Error("defaultConfirmUpdate non-TTY: returned false; want true (proceed without prompting)")
	}
}

// TestUpdateNonTTY_ProceedsWithoutPrompt verifies that a non-TTY (scripted)
// invocation proceeds to install via the Dispatch path. The confirmUpdate seam
// is stubbed to return (true, nil) — matching what defaultConfirmUpdate does
// when stdin is not a TTY — so the test focuses on end-to-end gate behavior
// rather than on the TTY-detection logic itself (which is covered by
// TestDefaultConfirmUpdate_NonTTY_ProceedsWithoutPrompt above).
func TestUpdateNonTTY_ProceedsWithoutPrompt(t *testing.T) {
	srv := makeUpdateSrvNoAssets(t, "v2.0.0")
	defer srv.Close()
	upd := makeUpdClientInternal(t, srv)

	withConfirmUpdate(t, func(current, tag string, force, versionTargetSet bool) (bool, error) {
		// Simulate non-TTY: proceed silently (same as defaultConfirmUpdate does).
		return true, nil
	})

	old := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	old2 := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	code := Dispatch([]string{"portkey", "update"}, "v1.0.0", "", upd)
	os.Stderr = old
	os.Stdout = old2

	// ExitRuntime (1) proves install was attempted — non-TTY did not block.
	if code != ExitRuntime {
		t.Errorf("non-TTY update: code = %d, want %d (install attempted)", code, ExitRuntime)
	}
}

// --- --check-only never prompts ---

// TestUpdateCheckOnly_NeverPrompts verifies confirmUpdate is NOT called when
// --check-only is set, regardless of other flags.
func TestUpdateCheckOnly_NeverPrompts(t *testing.T) {
	srv := makeUpdateSrvNoAssets(t, "v2.0.0")
	defer srv.Close()
	upd := makeUpdClientInternal(t, srv)

	rec := &confirmRecord{}
	withConfirmUpdate(t, func(current, tag string, force, versionTargetSet bool) (bool, error) {
		rec.called = true
		return false, nil
	})

	old := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	Dispatch([]string{"portkey", "update", "--check-only"}, "v1.0.0", "", upd)
	os.Stdout = old

	if rec.called {
		t.Error("confirmUpdate was called on --check-only; it must never be called on check-only paths")
	}
}

// TestUpdateDryRun_NeverPrompts is identical for --dry-run.
func TestUpdateDryRun_NeverPrompts(t *testing.T) {
	srv := makeUpdateSrvNoAssets(t, "v2.0.0")
	defer srv.Close()
	upd := makeUpdClientInternal(t, srv)

	rec := &confirmRecord{}
	withConfirmUpdate(t, func(current, tag string, force, versionTargetSet bool) (bool, error) {
		rec.called = true
		return false, nil
	})

	old := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	Dispatch([]string{"portkey", "update", "--dry-run"}, "v1.0.0", "", upd)
	os.Stdout = old

	if rec.called {
		t.Error("confirmUpdate was called on --dry-run; it must never be called on check-only paths")
	}
}

// --- --version-target install path is gated ---

// TestUpdateVersionTarget_IsGated verifies that --version-target goes through
// the confirmation gate (deny → Canceled. + ExitSuccess).
func TestUpdateVersionTarget_IsGated(t *testing.T) {
	srv := makeUpdateSrvNoAssets(t, "v0.5.0")
	defer srv.Close()
	upd := makeUpdClientInternal(t, srv)

	withConfirmUpdate(t, func(current, tag string, force, versionTargetSet bool) (bool, error) {
		return false, nil // deny
	})

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	code := Dispatch([]string{"portkey", "update", "--version-target", "v0.5.0"}, "v99.0.0", "", upd)
	w.Close()
	os.Stdout = old

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if code != ExitSuccess {
		t.Errorf("--version-target deny: code = %d, want %d (ExitSuccess)", code, ExitSuccess)
	}
	if !strings.Contains(output, "Canceled.") {
		t.Errorf("--version-target deny: output = %q, want 'Canceled.'", output)
	}
}

// --- --force install path is gated ---

// TestUpdateForce_IsGated verifies that --force goes through the confirmation
// gate (deny → Canceled. + ExitSuccess).
func TestUpdateForce_IsGated(t *testing.T) {
	srv := makeUpdateSrvNoAssets(t, "v1.0.0")
	defer srv.Close()
	upd := makeUpdClientInternal(t, srv)

	withConfirmUpdate(t, func(current, tag string, force, versionTargetSet bool) (bool, error) {
		return false, nil // deny
	})

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	code := Dispatch([]string{"portkey", "update", "--force"}, "v1.0.0", "", upd)
	w.Close()
	os.Stdout = old

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if code != ExitSuccess {
		t.Errorf("--force deny: code = %d, want %d (ExitSuccess)", code, ExitSuccess)
	}
	if !strings.Contains(output, "Canceled.") {
		t.Errorf("--force deny: output = %q, want 'Canceled.'", output)
	}
}
