package updater

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
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

// --- helpers ---

// tarEntry pairs a tar header with its optional payload bytes.
type tarEntry struct {
	hdr     *tar.Header
	payload []byte
}

// buildTarGz builds a tar.gz archive from the provided entries in order.
// payload may be nil for non-regular entries (symlink, dir, hardlink).
func buildTarGz(t *testing.T, entries []tarEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		if err := tw.WriteHeader(e.hdr); err != nil {
			t.Fatalf("write header %q: %v", e.hdr.Name, err)
		}
		if e.payload != nil {
			if _, err := tw.Write(e.payload); err != nil {
				t.Fatalf("write payload %q: %v", e.hdr.Name, err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	return buf.Bytes()
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

// --- Issue #52: extractBinary type / path / size guards ---

// TestExtractBinary_RejectsSymlink verifies a tar entry of type symlink named
// "portkey" is rejected rather than followed/read. Previously any non-dir
// entry named portkey was accepted, allowing symlink-based attacks (#52).
func TestExtractBinary_RejectsSymlink(t *testing.T) {
	entries := []tarEntry{{
		hdr: &tar.Header{
			Name:     "portkey",
			Typeflag: tar.TypeSymlink,
			Linkname: "/etc/passwd",
			Mode:     0o777,
		},
	}}
	archive := buildTarGz(t, entries)

	_, err := extractBinary(bytes.NewReader(archive))
	if err == nil {
		t.Fatal("extractBinary() error = nil, want error for symlink entry named portkey")
	}
}

// TestExtractBinary_RejectsHardlink verifies a tar hardlink entry named
// "portkey" is rejected.
func TestExtractBinary_RejectsHardlink(t *testing.T) {
	entries := []tarEntry{
		{
			hdr: &tar.Header{
				Name:     "target",
				Typeflag: tar.TypeReg,
				Mode:     0o644,
				Size:     int64(len("data")),
			},
			payload: []byte("data"),
		},
		{
			hdr: &tar.Header{
				Name:     "portkey",
				Typeflag: tar.TypeLink,
				Linkname: "target",
				Mode:     0o755,
			},
		},
	}
	archive := buildTarGz(t, entries)

	_, err := extractBinary(bytes.NewReader(archive))
	if err == nil {
		t.Fatal("extractBinary() error = nil, want error for hardlink entry named portkey")
	}
}

// TestExtractBinary_RejectsOversizedEntry verifies an entry larger than
// maxBinarySize is rejected before io.ReadAll can allocate the full size
// (decompression-bomb / OOM guard, #52). The archive is streamed so the test
// itself doesn't allocate 100 MiB.
func TestExtractBinary_RejectsOversizedEntry(t *testing.T) {
	huge := int64(maxBinarySize + 1)

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	hdr := &tar.Header{
		Name:     "portkey",
		Typeflag: tar.TypeReg,
		Mode:     0o755,
		Size:     huge,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("write header: %v", err)
	}
	// Stream exactly huge bytes so the tar is internally consistent; the
	// guard must trip on the header size and never reach the end of this.
	if _, err := io.CopyN(tw, zeroReader{}, huge); err != nil {
		t.Fatalf("stream payload: %v", err)
	}
	tw.Close()
	gz.Close()

	_, err := extractBinary(bytes.NewReader(buf.Bytes()))
	if err == nil {
		t.Fatal("extractBinary() error = nil, want error for oversized entry")
	}
}

// zeroReader is an endless source of zero bytes for streaming large payloads.
type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}

// TestExtractBinary_RejectsPathTraversal verifies a regular file entry whose
// name contains path separators / ".." components is rejected, so an attacker
// can't smuggle a name that escapes the base-name check (#52).
func TestExtractBinary_RejectsPathTraversal(t *testing.T) {
	cases := []string{
		"../portkey",
		"./portkey",
		"subdir/portkey",
		"/portkey",
	}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			payload := []byte("binary")
			entries := []tarEntry{{
				hdr: &tar.Header{
					Name:     name,
					Typeflag: tar.TypeReg,
					Mode:     0o755,
					Size:     int64(len(payload)),
				},
				payload: payload,
			}}
			archive := buildTarGz(t, entries)

			_, err := extractBinary(bytes.NewReader(archive))
			if err == nil {
				t.Fatalf("extractBinary() error = nil for name %q, want error", name)
			}
		})
	}
}

// TestExtractBinary_AcceptsValidEntry verifies a plain regular file named
// exactly "portkey" (no separators) is still extracted correctly — the guards
// didn't tighten the happy path out of existence.
func TestExtractBinary_AcceptsValidEntry(t *testing.T) {
	payload := []byte("REAL-BINARY")
	entries := []tarEntry{{
		hdr: &tar.Header{
			Name:     "portkey",
			Typeflag: tar.TypeReg,
			Mode:     0o755,
			Size:     int64(len(payload)),
		},
		payload: payload,
	}}
	archive := buildTarGz(t, entries)

	got, err := extractBinary(bytes.NewReader(archive))
	if err != nil {
		t.Fatalf("extractBinary() error = %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("got %q, want %q", got, payload)
	}
}

// --- Issue #52: unbounded read caps (downloadBytes / verifyChecksum) ---

// oversizedBodyServer serves a response body that streams maxBinarySize+1 zero
// bytes without keeping them all in memory at once on the server side.
func oversizedBodyServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		// Stream in chunks so the server itself doesn't allocate the cap.
		chunk := make([]byte, 64*1024)
		sent := int64(0)
		target := int64(maxBinarySize) + 1
		for sent < target {
			n := int64(len(chunk))
			if sent+n > target {
				n = target - sent
			}
			if _, err := w.Write(chunk[:n]); err != nil {
				return
			}
			sent += n
		}
	}))
}

// TestDownloadBytes_CapsBody verifies downloadBytes rejects a body larger than
// maxBinarySize instead of unbounded io.ReadAll (decompression-bomb, #52).
func TestDownloadBytes_CapsBody(t *testing.T) {
	srv := oversizedBodyServer(t)
	defer srv.Close()

	c := &Client{HTTP: srv.Client()}

	_, err := c.downloadBytes(srv.URL)
	if err == nil {
		t.Fatal("downloadBytes() error = nil, want error for oversized body")
	}
}

// TestVerifyChecksum_CapsBody verifies verifyChecksum rejects a checksums body
// larger than maxBinarySize.
func TestVerifyChecksum_CapsBody(t *testing.T) {
	srv := oversizedBodyServer(t)
	defer srv.Close()

	c := &Client{HTTP: srv.Client()}
	assets := []Asset{{Name: "checksums.txt", URL: srv.URL}}

	err := c.verifyChecksum(assets, "portkey.tar.gz", []byte("x"))
	if err == nil {
		t.Fatal("verifyChecksum() error = nil, want error for oversized checksum body")
	}
}
