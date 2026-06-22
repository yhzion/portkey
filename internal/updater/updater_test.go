package updater

import (
	"net/http"
	"net/http/httptest"
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
	if _, err := c.CheckLatest(); err != nil {
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
	_, err := c.CheckLatest()
	if err == nil {
		t.Fatal("CheckLatest() error = nil, want rate limited")
	}
	if err.Error() != "rate limited" {
		t.Errorf("error = %q, want %q", err.Error(), "rate limited")
	}
}
