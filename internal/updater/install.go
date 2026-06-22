package updater

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// osRename is os.Rename as a variable so tests can inject a failing rename to
// simulate cross-device (EXDEV) failures (issue #51).
var osRename = os.Rename

// DownloadAndInstall downloads the release asset for the current platform,
// verifies its SHA-256 checksum against checksums.txt, extracts the portkey
// binary, and replaces the currently running executable.
func (c *Client) DownloadAndInstall(rel *Release) error {
	asset, ok := CurrentAsset(rel.Assets)
	if !ok {
		return fmt.Errorf("no binary available for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	tarball, err := c.downloadBytes(asset.URL)
	if err != nil {
		return fmt.Errorf("download %s: %w", asset.Name, err)
	}

	if err := c.verifyChecksum(rel.Assets, asset.Name, tarball); err != nil {
		return fmt.Errorf("checksum: %w", err)
	}

	bin, err := extractBinary(bytes.NewReader(tarball))
	if err != nil {
		return fmt.Errorf("extract: %w", err)
	}

	exePath, err := currentExe()
	if err != nil {
		return err
	}

	if err := replaceFile(exePath, bin); err != nil {
		return fmt.Errorf("install: %w", err)
	}

	return nil
}

// downloadBytes fetches a URL and returns the full response body.
func (c *Client) downloadBytes(url string) ([]byte, error) {
	resp, err := c.HTTP.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// verifyChecksum downloads checksums.txt from the release assets and verifies
// that the SHA-256 of data matches the entry for name.
func (c *Client) verifyChecksum(assets []Asset, name string, data []byte) error {
	var checksumURL string
	for _, a := range assets {
		if a.Name == "checksums.txt" {
			checksumURL = a.URL
			break
		}
	}
	if checksumURL == "" {
		return fmt.Errorf("checksums.txt not found in release")
	}

	resp, err := c.HTTP.Get(checksumURL)
	if err != nil {
		return fmt.Errorf("fetch checksums: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch checksums: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read checksums: %w", err)
	}

	expected := findHash(string(body), name)
	if expected == "" {
		return fmt.Errorf("no checksum entry for %s", name)
	}

	actual := fmt.Sprintf("%x", sha256.Sum256(data))
	if actual != expected {
		return fmt.Errorf(
			"mismatch for %s: expected %s, got %s",
			name, expected, actual,
		)
	}

	return nil
}

// findHash parses checksums.txt content and returns the hash for filename.
// Line format: "<sha256>  <filename>"
func findHash(content, filename string) string {
	for _, line := range strings.Split(content, "\n") {
		parts := strings.SplitN(line, "  ", 2)
		if len(parts) == 2 && parts[1] == filename {
			return parts[0]
		}
	}
	return ""
}

// extractBinary decompresses a tar.gz archive and returns the portkey binary.
func extractBinary(r io.Reader) ([]byte, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("tar: %w", err)
		}
		if filepath.Base(hdr.Name) == "portkey" && hdr.Typeflag != tar.TypeDir {
			data, err := io.ReadAll(tr)
			if err != nil {
				return nil, fmt.Errorf("read portkey: %w", err)
			}
			return data, nil
		}
	}

	return nil, fmt.Errorf("portkey binary not found in archive")
}

// currentExe returns the resolved path of the running executable.
func currentExe() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("get executable path: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("resolve symlinks: %w", err)
	}
	return resolved, nil
}

// replaceFile atomically replaces the file at path with content.
func replaceFile(path string, content []byte) error {
	fi, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat current binary: %w", err)
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".portkey-update-*")
	if err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf(
				"permission denied — try with appropriate privileges",
			)
		}
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	cleanup := func() {
		tmp.Close()
		os.Remove(tmpPath)
	}

	if _, err := tmp.Write(content); err != nil {
		cleanup()
		return fmt.Errorf("write binary: %w", err)
	}

	if err := tmp.Chmod(fi.Mode()); err != nil {
		cleanup()
		return fmt.Errorf("set permissions: %w", err)
	}

	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := osRename(tmpPath, path); err != nil {
		// Rename failed — most commonly because the install directory is on a
		// different filesystem than the temp directory (EXDEV). Fall back to a
		// safe replace: back up the live binary to a .old file first so the
		// user can recover if anything below dies, then write the new content
		// to a second temp file in the SAME directory as the target and rename
		// it into place (atomic, since it's within one filesystem). Only if
		// that rename also fails do we byte-copy — but the .old backup is
		// already on disk. We never O_TRUNC the live binary directly, so a
		// crash cannot leave a truncated/corrupt binary (issue #51).
		fallbackErr := copyAtomic(tmpPath, path, fi.Mode())
		os.Remove(tmpPath)
		if fallbackErr != nil {
			return fmt.Errorf("rename: %v; atomic fallback: %w", err, fallbackErr)
		}
	}

	return nil
}

// copyAtomic replaces dst with the contents of src using a same-directory
// rename, leaving a ".old" backup of the previous dst so the user can recover
// if the process dies mid-update. It never opens dst with O_TRUNC, so a crash
// cannot corrupt the live binary (issue #51).
//
// On success dst holds the new content and dst+".old" holds the previous
// content. On failure dst is left untouched.
func copyAtomic(src, dst string, mode os.FileMode) error {
	dir := filepath.Dir(dst)

	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open src: %w", err)
	}
	defer in.Close()

	// Stage the new content in a temp file in the DESTINATION directory so the
	// final rename is within one filesystem (atomic).
	tmp, err := os.CreateTemp(dir, ".portkey-replace-*")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()

	staged := false
	defer func() {
		if !staged {
			tmp.Close()
			os.Remove(tmpPath)
		}
	}()

	if _, err := io.Copy(tmp, in); err != nil {
		return fmt.Errorf("stage copy: %w", err)
	}
	if err := tmp.Chmod(mode); err != nil {
		return fmt.Errorf("chmod temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp: %w", err)
	}

	// Back up the current binary so the user can recover on total failure.
	// Remove any stale .old first (rename won't overwrite on all platforms).
	oldPath := dst + ".old"
	_ = os.Remove(oldPath)
	if err := os.Rename(dst, oldPath); err != nil {
		// If the backup rename fails we abort — better to leave the live
		// binary intact than to proceed without a recovery path.
		return fmt.Errorf("backup current binary: %w", err)
	}

	// Atomic rename within the destination directory. This typically succeeds
	// where the original cross-device rename failed because tmpPath lives in
	// the same directory as dst.
	if err := os.Rename(tmpPath, dst); err != nil {
		// Rename failed after the backup: restore the original immediately.
		_ = os.Rename(oldPath, dst)
		return fmt.Errorf("rename into place: %w", err)
	}
	staged = true
	return nil
}
