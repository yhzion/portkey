package updater

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestVerifyChecksumNon200 verifies that a non-200 response when fetching
// checksums.txt is surfaced as an error mentioning the status rather than
// silently succeeding (MITM / outage hardening).
func TestVerifyChecksumNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := &Client{HTTP: srv.Client()}
	assets := []Asset{{Name: "checksums.txt", URL: srv.URL}}

	err := c.verifyChecksum(assets, "portkey_0.1.0_linux_amd64.tar.gz", []byte("data"))
	if err == nil {
		t.Fatal("verifyChecksum() error = nil, want error for non-200 checksums response")
	}
	if !strings.Contains(err.Error(), "status") {
		t.Errorf("error = %q, want it to mention the status", err.Error())
	}
}

// writeBinary writes content to path with the given mode, failing the test on
// error.
func writeBinary(t *testing.T, path string, content []byte, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, content, mode); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatalf("chmod %s: %v", path, err)
	}
}

// --- Issue #51: atomic self-replace ---

// TestReplaceFile_PreservesBinaryOnRenameFailure verifies that when the rename
// step fails (e.g. cross-device / EXDEV), the destination binary is NOT
// truncated and a .old backup is left behind so the user can recover.
// Previously the fallback opened dst with O_TRUNC and copied in place, so a
// mid-copy crash left a corrupt binary with no backup (issue #51).
func TestReplaceFile_PreservesBinaryOnRenameFailure(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "portkey")

	original := []byte("ORIGINAL-BINARY")
	writeBinary(t, dst, original, 0o755)

	newContent := []byte("NEW-BINARY")

	// Force rename to fail to simulate a cross-device / EXDEV scenario.
	prevRename := osRename
	t.Cleanup(func() { osRename = prevRename })
	osRename = func(oldpath, newpath string) error {
		return errors.New("simulated cross-device rename")
	}

	err := replaceFile(dst, newContent)
	if err != nil {
		t.Fatalf("replaceFile() with failed rename should fall back and succeed, got error: %v", err)
	}

	// The live binary MUST now be the new content (fallback succeeded) ...
	got, readErr := os.ReadFile(dst)
	if readErr != nil {
		t.Fatalf("read dst after failed replace: %v", readErr)
	}
	if !bytes.Equal(got, newContent) {
		t.Errorf("dst = %q, want %q (fallback must install new content)", got, newContent)
	}

	// A .old backup must exist so the user can recover.
	backup := dst + ".old"
	backupContent, err := os.ReadFile(backup)
	if err != nil {
		t.Fatalf("expected backup at %s, got error: %v", backup, err)
	}
	if !bytes.Equal(backupContent, original) {
		t.Errorf("backup = %q, want original %q", backupContent, original)
	}
}

// TestReplaceFile_DoesNotTruncateOnTotalFailure verifies that if BOTH rename
// and the copy fallback fail, the destination binary is left completely intact
// (not truncated/corrupt). This is the corruption-on-crash scenario from #51.
func TestReplaceFile_DoesNotTruncateOnTotalFailure(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "portkey")

	original := []byte("ORIGINAL-BINARY")
	writeBinary(t, dst, original, 0o755)

	// Make rename fail.
	prevRename := osRename
	t.Cleanup(func() { osRename = prevRename })
	osRename = func(oldpath, newpath string) error {
		return errors.New("simulated cross-device rename")
	}

	// Revoke write permission on the dir so the backup / fallback also fails
	// (simulating a worst-case crash mid-fallback). Restore perms on cleanup.
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("chmod dir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	_ = replaceFile(dst, []byte("NEW"))

	// Restore perms so we can read.
	_ = os.Chmod(dir, 0o755)

	// The live binary MUST be intact regardless of fallback outcome.
	got, readErr := os.ReadFile(dst)
	if readErr != nil {
		t.Fatalf("read dst after total failure: %v", readErr)
	}
	if !bytes.Equal(got, original) {
		t.Errorf("dst = %q, want intact %q (must never be truncated on failure)", got, original)
	}
}

// TestReplaceFile_AtomicRenameSuccess verifies the happy path still replaces
// the binary in place when rename succeeds, and leaves no stray temp files.
func TestReplaceFile_AtomicRenameSuccess(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "portkey")

	writeBinary(t, dst, []byte("OLD"), 0o755)

	newContent := []byte("NEW")
	if err := replaceFile(dst, newContent); err != nil {
		t.Fatalf("replaceFile() error = %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if !bytes.Equal(got, newContent) {
		t.Errorf("dst = %q, want %q", got, newContent)
	}

	// No stray temp files or .old backup when the atomic rename worked.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, e := range entries {
		if e.Name() != "portkey" {
			t.Errorf("unexpected leftover entry %q in dir", e.Name())
		}
	}
}

// TestReplaceFile_PreservesPermissions verifies the replaced binary keeps the
// original file mode (executable).
func TestReplaceFile_PreservesPermissions(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "portkey")

	writeBinary(t, dst, []byte("OLD"), 0o755)

	if err := replaceFile(dst, []byte("NEW")); err != nil {
		t.Fatalf("replaceFile() error = %v", err)
	}

	fi, err := os.Stat(dst)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if fi.Mode().Perm() != 0o755 {
		t.Errorf("mode = %o, want 755", fi.Mode().Perm())
	}
}
