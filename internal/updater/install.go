package updater

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// maxBinarySize caps how many bytes are read from the download, checksum file,
// and extracted tar entry. A real portkey build is a few MiB; this ceiling
// guards against decompression bombs / OOM during an auto-update (issue #52).
const maxBinarySize = 100 << 20 // 100 MiB

// osRename is os.Rename as a variable so tests can inject a failing rename to
// simulate cross-device (EXDEV) failures (issue #51).
var osRename = os.Rename

// osExecutable is os.Executable as a variable so tests can inject a fake path
// to avoid touching the real running binary.
var osExecutable = os.Executable

// DownloadAndInstall downloads the release asset for the current platform,
// verifies its SHA-256 checksum against checksums.txt, extracts the portkey
// binary, and replaces the currently running executable.
//
// progress is an optional callback invoked before each phase with a short
// human-readable name ("Downloading", "Verifying checksum", "Installing").
// Pass nil for silent operation (preserves the prior behaviour).
func (c *Client) DownloadAndInstall(rel *Release, progress func(phase string)) error {
	asset, ok := CurrentAsset(rel.Assets)
	if !ok {
		return fmt.Errorf("no binary available for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	if progress != nil {
		progress("Downloading")
	}
	tarball, err := c.downloadBytes(asset.URL)
	if err != nil {
		return fmt.Errorf("download %s: %w", asset.Name, err)
	}

	if progress != nil {
		progress("Verifying checksum")
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

	if progress != nil {
		progress("Installing")
	}
	if err := replaceFile(exePath, bin); err != nil {
		return fmt.Errorf("install: %w", err)
	}

	return nil
}

// downloadBytes fetches a URL and returns the full response body.
// It uses c.DownloadHTTP (no fixed timeout) so multi-MB binary downloads are
// not cut off by the short check-client timeout.
func (c *Client) downloadBytes(url string) ([]byte, error) {
	dlClient := c.DownloadHTTP
	if dlClient == nil {
		dlClient = c.HTTP
	}
	resp, err := dlClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	// Cap the read so a malicious / oversized asset can't exhaust memory (#52).
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBinarySize+1))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if int64(len(data)) > maxBinarySize {
		return nil, fmt.Errorf("download exceeds %d bytes", maxBinarySize)
	}
	return data, nil
}

// errAssetNotFound indicates a named asset is absent from the release.
var errAssetNotFound = errors.New("asset not found in release")

// downloadAsset finds the named asset in the release and returns its body,
// status-checked and size-capped. Returns errAssetNotFound (wrapped) when the
// asset name is not present.
func (c *Client) downloadAsset(assets []Asset, name string) ([]byte, error) {
	var url string
	for _, a := range assets {
		if a.Name == name {
			url = a.URL
			break
		}
	}
	if url == "" {
		return nil, fmt.Errorf("%s: %w", name, errAssetNotFound)
	}
	dlClient := c.DownloadHTTP
	if dlClient == nil {
		dlClient = c.HTTP
	}
	resp, err := dlClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: status %d", name, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBinarySize+1))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", name, err)
	}
	if int64(len(body)) > maxBinarySize {
		return nil, fmt.Errorf("%s exceeds %d bytes", name, maxBinarySize)
	}
	return body, nil
}

// verifyChecksum downloads checksums.txt from the release assets, verifies the
// minisign signature over it (fail-closed: missing or invalid signature aborts
// the install), and then checks that the SHA-256 of data matches the entry for
// name.
func (c *Client) verifyChecksum(assets []Asset, name string, data []byte) error {
	body, err := c.downloadAsset(assets, "checksums.txt")
	if err != nil {
		return fmt.Errorf("checksums: %w", err)
	}

	// Fail-closed signature verification: the release MUST ship a valid
	// checksums.txt.minisig signed by the pinned key. A missing or invalid
	// signature aborts the install.
	sig, err := c.downloadAsset(assets, "checksums.txt.minisig")
	if err != nil {
		if errors.Is(err, errAssetNotFound) {
			return fmt.Errorf("release is not signed (checksums.txt.minisig missing); refusing to install")
		}
		return fmt.Errorf("signature: %w", err)
	}
	if err := verifyMinisign(c.signPubKey, body, sig); err != nil {
		return fmt.Errorf("signature verification failed; refusing to install: %w", err)
	}

	expected := findHash(string(body), name)
	if expected == "" {
		return fmt.Errorf("no checksum entry for %s", name)
	}
	actual := fmt.Sprintf("%x", sha256.Sum256(data))
	if actual != expected {
		return fmt.Errorf("mismatch for %s: expected %s, got %s", name, expected, actual)
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
		// Only accept a regular file named exactly "portkey" — no path
		// separators, no ".." traversal, and never symlink/hardlink/dir entries
		// (issue #52).
		if hdr.Name != "portkey" || filepath.Base(hdr.Name) != hdr.Name {
			continue
		}
		if hdr.Typeflag != tar.TypeReg {
			return nil, fmt.Errorf(
				"portkey entry is not a regular file (type %d)", hdr.Typeflag,
			)
		}
		if hdr.Size > maxBinarySize {
			return nil, fmt.Errorf(
				"portkey entry size %d exceeds %d bytes", hdr.Size, maxBinarySize,
			)
		}
		data, err := io.ReadAll(io.LimitReader(tr, maxBinarySize+1))
		if err != nil {
			return nil, fmt.Errorf("read portkey: %w", err)
		}
		if int64(len(data)) > maxBinarySize {
			return nil, fmt.Errorf(
				"portkey entry exceeds %d bytes", maxBinarySize,
			)
		}
		return data, nil
	}

	return nil, fmt.Errorf("portkey binary not found in archive")
}

// currentExe returns the resolved path of the running executable.
// It uses osExecutable so tests can inject a fake path.
func currentExe() (string, error) {
	exe, err := osExecutable()
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
