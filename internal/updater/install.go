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

	if err := os.Rename(tmpPath, path); err != nil {
		if copyErr := copyContent(tmpPath, path); copyErr != nil {
			os.Remove(tmpPath)
			return fmt.Errorf(
				"rename: %v; copy fallback: %w",
				err, copyErr,
			)
		}
		os.Remove(tmpPath)
	}

	return nil
}

// copyContent overwrites dst with the contents of src.
func copyContent(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
