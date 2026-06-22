package updater

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"testing"
)

// --- CheckRelease ---

// TestCheckRelease_HitsTagsPath verifies that CheckRelease requests
// /releases/tags/{tag} (not /releases/latest).
func TestCheckRelease_HitsTagsPath(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ghRelease{TagName: "v0.9.0", Assets: []ghAsset{}})
	}))
	defer srv.Close()

	c := &Client{HTTP: srv.Client(), Owner: "yhzion", Repo: "portkey", BaseURL: srv.URL}
	rel, err := c.CheckRelease(context.Background(), "v0.9.0")
	if err != nil {
		t.Fatalf("CheckRelease() error = %v", err)
	}
	if rel.Tag != "v0.9.0" {
		t.Errorf("Tag = %q, want %q", rel.Tag, "v0.9.0")
	}
	want := "/repos/yhzion/portkey/releases/tags/v0.9.0"
	if gotPath != want {
		t.Errorf("request path = %q, want %q", gotPath, want)
	}
}

// TestCheckRelease_SendsUserAgent verifies CheckRelease sends User-Agent.
func TestCheckRelease_SendsUserAgent(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ghRelease{TagName: "v0.9.0"})
	}))
	defer srv.Close()

	c := &Client{HTTP: srv.Client(), Owner: "o", Repo: "r", BaseURL: srv.URL}
	if _, err := c.CheckRelease(context.Background(), "v0.9.0"); err != nil {
		t.Fatalf("CheckRelease() error = %v", err)
	}
	if gotUA == "" {
		t.Error("User-Agent header was empty, want non-empty")
	}
}

// TestCheckRelease_404_NotFound verifies that a 404 for a specific tag
// wraps ErrNoReleases (mirrors CheckLatest behaviour).
func TestCheckRelease_404_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := &Client{HTTP: srv.Client(), Owner: "o", Repo: "r", BaseURL: srv.URL}
	_, err := c.CheckRelease(context.Background(), "v0.9.0")
	if err == nil {
		t.Fatal("CheckRelease() error = nil, want error for 404")
	}
	if !errors.Is(err, ErrNoReleases) {
		t.Errorf("error should wrap ErrNoReleases, got %q", err.Error())
	}
}

// TestCheckRelease_RateLimited verifies 429 wraps ErrRateLimited.
func TestCheckRelease_RateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := &Client{HTTP: srv.Client(), Owner: "o", Repo: "r", BaseURL: srv.URL}
	_, err := c.CheckRelease(context.Background(), "v0.9.0")
	if err == nil {
		t.Fatal("CheckRelease() error = nil, want error for 429")
	}
	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("error should wrap ErrRateLimited, got %q", err.Error())
	}
}

// --- Tag validation ---

// TestValidateTag_Rejects verifies that invalid tags are rejected BEFORE
// any HTTP request is made.
func TestValidateTag_Rejects(t *testing.T) {
	cases := []struct {
		tag  string
		desc string
	}{
		{"", "empty"},
		{"/", "bare slash"},
		{"a/b", "slash in middle"},
		{"..", "double dot"},
		{"a..b", "double dot embedded"},
		{"../etc", "path traversal"},
		{"v1.0 0", "space"},
		{"v1.0\t0", "tab"},
		{"v1.0\n0", "newline"},
		{"\x00v1", "null byte"},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			// Use a server that would error if reached, proving the rejection
			// happens before any network call.
			networkCalled := false
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				networkCalled = true
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			c := &Client{HTTP: srv.Client(), Owner: "o", Repo: "r", BaseURL: srv.URL}
			_, err := c.CheckRelease(context.Background(), tc.tag)
			if err == nil {
				t.Errorf("CheckRelease(%q) error = nil, want error for invalid tag", tc.tag)
			}
			if networkCalled {
				t.Errorf("network was called for invalid tag %q; rejection must be pre-flight", tc.tag)
			}
		})
	}
}

// TestValidateTag_Accepts verifies version-ish tags are accepted.
func TestValidateTag_Accepts(t *testing.T) {
	cases := []string{
		"v1.0.0",
		"v1.2.3-rc1",
		"v1.0.0+build5",
		"v1.0.0-rc1+build5",
		"0.1.0",
		"v10.20.30",
	}

	for _, tag := range cases {
		t.Run(tag, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(ghRelease{TagName: tag})
			}))
			defer srv.Close()

			c := &Client{HTTP: srv.Client(), Owner: "o", Repo: "r", BaseURL: srv.URL}
			_, err := c.CheckRelease(context.Background(), tag)
			if err != nil {
				t.Errorf("CheckRelease(%q) error = %v, want nil for valid tag", tag, err)
			}
		})
	}
}

// --- Progress callback for DownloadAndInstall ---

// makeValidRelease builds a Release whose assets are served by the given mux.
// It writes archive and checksums handlers under /asset and /checksums.
func makeValidRelease(t *testing.T, mux *http.ServeMux, binaryContent []byte) *Release {
	t.Helper()

	archiveData := buildTarGz(t, []tarEntry{{
		hdr: &tar.Header{
			Name:     "portkey",
			Typeflag: tar.TypeReg,
			Mode:     0o755,
			Size:     int64(len(binaryContent)),
		},
		payload: binaryContent,
	}})

	hash := fmt.Sprintf("%x", sha256.Sum256(archiveData))
	// Name the asset for the CURRENT platform so CurrentAsset (which matches a
	// _<goos>_<goarch>.tar.gz suffix) resolves it on every CI runner, not just
	// linux/amd64 — DownloadAndInstall picks the asset via runtime.GOOS/GOARCH.
	assetName := fmt.Sprintf("portkey_0.1.0_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	checksumLine := hash + "  " + assetName + "\n"

	mux.HandleFunc("/asset", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(archiveData)
	})
	mux.HandleFunc("/checksums", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(checksumLine))
	})

	return &Release{
		Tag: "v0.1.0",
		Assets: []Asset{
			{Name: assetName, URL: "/asset"}, // placeholder; patched per-test
			{Name: "checksums.txt", URL: "/checksums"},
		},
	}
}

// TestDownloadAndInstall_ProgressCallback verifies that the progress callback
// is invoked with the expected phases in order.
func TestDownloadAndInstall_ProgressCallback(t *testing.T) {
	binaryContent := []byte("FAKE-BINARY-PROGRESS")

	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	rel := makeValidRelease(t, mux, binaryContent)
	// Fix asset URLs to use the test server's address.
	rel.Assets[0].URL = srv.URL + "/asset"
	rel.Assets[1].URL = srv.URL + "/checksums"

	// Intercept the final install step: redirect the exe path into a temp file.
	dir := t.TempDir()
	dst := dir + "/portkey"
	if err := os.WriteFile(dst, []byte("OLD"), 0o755); err != nil {
		t.Fatalf("setup fake exe: %v", err)
	}

	prevExe := osExecutable
	t.Cleanup(func() { osExecutable = prevExe })
	osExecutable = func() (string, error) { return dst, nil }

	var phases []string
	progress := func(phase string) { phases = append(phases, phase) }

	c := &Client{
		HTTP:         srv.Client(),
		DownloadHTTP: srv.Client(),
		Owner:        "o",
		Repo:         "r",
		BaseURL:      srv.URL,
	}
	if err := c.DownloadAndInstall(rel, progress); err != nil {
		t.Fatalf("DownloadAndInstall() error = %v", err)
	}

	want := []string{"Downloading", "Verifying checksum", "Installing"}
	if len(phases) != len(want) {
		t.Fatalf("progress phases = %v, want %v", phases, want)
	}
	for i, p := range want {
		if phases[i] != p {
			t.Errorf("phase[%d] = %q, want %q", i, phases[i], p)
		}
	}
}

// TestDownloadAndInstall_NilProgress verifies that a nil callback is silent
// and still completes without panicking.
func TestDownloadAndInstall_NilProgress(t *testing.T) {
	binaryContent := []byte("FAKE-BINARY-NIL-PROG")

	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	rel := makeValidRelease(t, mux, binaryContent)
	rel.Assets[0].URL = srv.URL + "/asset"
	rel.Assets[1].URL = srv.URL + "/checksums"

	dir := t.TempDir()
	dst := dir + "/portkey"
	if err := os.WriteFile(dst, []byte("OLD"), 0o755); err != nil {
		t.Fatalf("setup fake exe: %v", err)
	}

	prevExe := osExecutable
	t.Cleanup(func() { osExecutable = prevExe })
	osExecutable = func() (string, error) { return dst, nil }

	c := &Client{
		HTTP:         srv.Client(),
		DownloadHTTP: srv.Client(),
		Owner:        "o",
		Repo:         "r",
		BaseURL:      srv.URL,
	}
	// nil progress — must not panic.
	if err := c.DownloadAndInstall(rel, nil); err != nil {
		t.Fatalf("DownloadAndInstall(nil progress) error = %v", err)
	}
}
