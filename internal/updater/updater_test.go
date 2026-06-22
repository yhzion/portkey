package updater

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseVersionTag(t *testing.T) {
	tests := []struct {
		input string
		major int
		minor int
		patch int
		pre   string
		ok    bool
	}{
		{"v0.1.0", 0, 1, 0, "", true},
		{"v1.2.3", 1, 2, 3, "", true},
		{"v10.20.30", 10, 20, 30, "", true},
		{"0.1.0", 0, 1, 0, "", true},
		{"v1.0.0-rc1", 1, 0, 0, "rc1", true},
		{"1.0.0-rc1", 1, 0, 0, "rc1", true},
		{"v1.0.0+build5", 1, 0, 0, "", true},
		{"v1.0.0-rc1+build5", 1, 0, 0, "rc1", true},
		{"v0.1", 0, 0, 0, "", false},
		{"not-a-version", 0, 0, 0, "", false},
		{"", 0, 0, 0, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			maj, min, pat, pre, ok := ParseVersion(tt.input)
			if ok != tt.ok {
				t.Errorf("ok = %v, want %v", ok, tt.ok)
			}
			if ok && (maj != tt.major || min != tt.minor || pat != tt.patch || pre != tt.pre) {
				t.Errorf("got %d.%d.%d-%q, want %d.%d.%d-%q", maj, min, pat, pre, tt.major, tt.minor, tt.patch, tt.pre)
			}
		})
	}
}

// TestParseVersionPreReleaseDistinct verifies a pre-release version is parsed
// distinctly from the same major.minor.patch without one (issue #48).
func TestParseVersionPreReleaseDistinct(t *testing.T) {
	_, _, _, rcPre, rcOK := ParseVersion("1.0.0-rc1")
	_, _, _, relPre, relOK := ParseVersion("1.0.0")
	if !rcOK || !relOK {
		t.Fatalf("both versions should parse: rc=%v rel=%v", rcOK, relOK)
	}
	if rcPre == relPre {
		t.Errorf("pre-release tag = %q, want distinct from release %q", rcPre, relPre)
	}
	if rcPre != "rc1" {
		t.Errorf("pre = %q, want %q", rcPre, "rc1")
	}
	if relPre != "" {
		t.Errorf("release pre = %q, want empty", relPre)
	}
}

func TestIsNewer(t *testing.T) {
	tests := []struct {
		current string
		latest  string
		want    bool
	}{
		{"v0.1.0", "v0.1.1", true},
		{"v0.1.0", "v0.2.0", true},
		{"v0.1.0", "v1.0.0", true},
		{"v0.1.0", "v0.1.0", false},
		{"v0.2.0", "v0.1.0", false},
		{"v1.0.0", "v0.9.9", false},
		{"v0.1.0", "not-a-version", false},
		{"dev", "v0.1.0", false},
		{"v1.0.0-rc1", "v1.0.0", true},
		{"v1.0.0", "v1.0.0-rc1", false},
		{"v1.0.0-rc1", "v1.0.0-rc2", true},
		{"v1.0.0-rc2", "v1.0.0-rc1", false},
		{"v1.0.0-rc1", "v1.0.0-rc1", false},
		{"v1.0.0-rc1", "v1.0.1", true},
	}

	for _, tt := range tests {
		t.Run(tt.current+"_vs_"+tt.latest, func(t *testing.T) {
			got := IsNewer(tt.current, tt.latest)
			if got != tt.want {
				t.Errorf("IsNewer(%q, %q) = %v, want %v", tt.current, tt.latest, got, tt.want)
			}
		})
	}
}

func TestPickAsset(t *testing.T) {
	assets := []Asset{
		{Name: "portkey_0.1.0_darwin_amd64.tar.gz", URL: "url1"},
		{Name: "portkey_0.1.0_darwin_arm64.tar.gz", URL: "url2"},
		{Name: "portkey_0.1.0_linux_amd64.tar.gz", URL: "url3"},
		{Name: "portkey_0.1.0_linux_arm64.tar.gz", URL: "url4"},
		{Name: "checksums.txt", URL: "url5"},
	}

	tests := []struct {
		goos   string
		goarch string
		name   string
		ok     bool
	}{
		{"darwin", "arm64", "portkey_0.1.0_darwin_arm64.tar.gz", true},
		{"linux", "amd64", "portkey_0.1.0_linux_amd64.tar.gz", true},
		{"windows", "amd64", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.goos+"_"+tt.goarch, func(t *testing.T) {
			got, ok := PickAsset(assets, tt.goos, tt.goarch)
			if ok != tt.ok {
				t.Errorf("ok = %v, want %v", ok, tt.ok)
				return
			}
			if ok && got.Name != tt.name {
				t.Errorf("Name = %q, want %q", got.Name, tt.name)
			}
		})
	}
}

// TestCheckLatestSendsUserAgent verifies CheckLatest sets a non-empty
// User-Agent header (issue #50: GitHub 403s UA-less requests).
func TestCheckLatestSendsUserAgent(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"tag_name":"v1.0.0","assets":[]}`))
	}))
	defer srv.Close()

	c := &Client{HTTP: srv.Client(), Owner: "o", Repo: "r", BaseURL: srv.URL}
	if _, err := c.CheckLatest(context.Background()); err != nil {
		t.Fatalf("CheckLatest() error = %v", err)
	}
	if gotUA == "" {
		t.Error("User-Agent header was empty, want non-empty")
	}
}

// TestCheckLatestRateLimitedOn429 verifies a 429 response is surfaced as
// rate-limited rather than the generic "unexpected status" error (issue #50).
func TestCheckLatestRateLimitedOn429(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := &Client{HTTP: srv.Client(), Owner: "o", Repo: "r", BaseURL: srv.URL}
	_, err := c.CheckLatest(context.Background())
	if err == nil {
		t.Fatal("CheckLatest() error = nil, want rate limited")
	}
	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("error %q should wrap ErrRateLimited", err.Error())
	}
}

// TestClassifyCheckError verifies that 403, 429, 404, and transport errors
// each classify to the expected CheckErrorKind (issue #65).
func TestClassifyCheckError(t *testing.T) {
	tests := []struct {
		name     string
		status   int // 0 means transport/offline error
		wantKind CheckErrorKind
	}{
		{"403_forbidden", http.StatusForbidden, KindOther},
		{"429_too_many_requests", http.StatusTooManyRequests, KindRateLimited},
		{"404_not_found", http.StatusNotFound, KindNotFound},
		{"offline", 0, KindOffline},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.status == 0 {
				// Simulate a transport/offline error by connecting to a closed server.
				c := &Client{HTTP: http.DefaultClient, Owner: "o", Repo: "r", BaseURL: "http://127.0.0.1:0"}
				_, err = c.CheckLatest(context.Background())
			} else {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.status)
				}))
				defer srv.Close()
				c := &Client{HTTP: srv.Client(), Owner: "o", Repo: "r", BaseURL: srv.URL}
				_, err = c.CheckLatest(context.Background())
			}
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			got := ClassifyCheckError(err)
			if got != tt.wantKind {
				t.Errorf("ClassifyCheckError(%q) = %v, want %v", err.Error(), got, tt.wantKind)
			}
			// Wrapping robustness: an additional layer of wrapping must not break
			// classification (proves errors.Is-based sentinel dispatch works through
			// arbitrary wrapping, e.g. "update check: %w" then "context: %w").
			wrapped := fmt.Errorf("outer context: %w", err)
			if got2 := ClassifyCheckError(wrapped); got2 != tt.wantKind {
				t.Errorf("ClassifyCheckError(wrapped %q) = %v, want %v", wrapped.Error(), got2, tt.wantKind)
			}
		})
	}
}

// TestCheckLatest_404_NotFound verifies that a 404 response is treated as
// "no releases yet" (NotFound) rather than "unexpected status: 404" (issue #65).
func TestCheckLatest_404_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := &Client{HTTP: srv.Client(), Owner: "o", Repo: "r", BaseURL: srv.URL}
	_, err := c.CheckLatest(context.Background())
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
	if strings.Contains(err.Error(), "unexpected status") {
		t.Errorf("404 should not produce 'unexpected status' error, got: %q", err.Error())
	}
}

// TestCheckLatest_403_and_429_DistinctMessages verifies that 403 and 429 produce
// distinct error messages (issue #65).
func TestCheckLatest_403_and_429_DistinctMessages(t *testing.T) {
	makeClient := func(status int) (*Client, func()) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
		}))
		return &Client{HTTP: srv.Client(), Owner: "o", Repo: "r", BaseURL: srv.URL}, srv.Close
	}

	c403, close403 := makeClient(http.StatusForbidden)
	defer close403()
	_, err403 := c403.CheckLatest(context.Background())
	if err403 == nil {
		t.Fatal("expected error for 403")
	}

	c429, close429 := makeClient(http.StatusTooManyRequests)
	defer close429()
	_, err429 := c429.CheckLatest(context.Background())
	if err429 == nil {
		t.Fatal("expected error for 429")
	}

	if err403.Error() == err429.Error() {
		t.Errorf("403 and 429 should produce distinct errors; both = %q", err403.Error())
	}
}
