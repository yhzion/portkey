package updater

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"
)

// Asset represents a GitHub release asset.
type Asset struct {
	Name string
	URL  string
}

// Release represents a GitHub release with its tag and assets.
type Release struct {
	Tag    string
	Assets []Asset
}

// ParseVersion parses "vX.Y.Z", "X.Y.Z", or a semver string with an optional
// pre-release tag ("X.Y.Z-rc1") and/or build metadata ("X.Y.Z+build") into
// major, minor, patch and the pre-release identifier. Build metadata is
// discarded. The returned pre is "" when no pre-release tag is present.
func ParseVersion(tag string) (maj, min, pat int, pre string, ok bool) {
	tag = strings.TrimPrefix(tag, "v")
	// Strip build metadata (everything after the first '+'); it is ignored
	// for precedence.
	if i := strings.IndexByte(tag, '+'); i >= 0 {
		tag = tag[:i]
	}
	// Split off the pre-release tag (everything after the first '-').
	if i := strings.IndexByte(tag, '-'); i >= 0 {
		pre = tag[i+1:]
		tag = tag[:i]
	}
	parts := strings.SplitN(tag, ".", 3)
	if len(parts) != 3 {
		return 0, 0, 0, "", false
	}
	if _, err := fmt.Sscanf(parts[0], "%d", &maj); err != nil {
		return 0, 0, 0, "", false
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &min); err != nil {
		return 0, 0, 0, "", false
	}
	if _, err := fmt.Sscanf(parts[2], "%d", &pat); err != nil {
		return 0, 0, 0, "", false
	}
	return maj, min, pat, pre, true
}

// IsNewer returns true if latest > current (semver comparison).
// Returns false if either version is unparseable. A version with a pre-release
// tag ranks lower than the same major.minor.patch without one, so a user on a
// pre-release is offered the matching final release.
func IsNewer(current, latest string) bool {
	cmaj, cmin, cpat, cpre, ok := ParseVersion(current)
	if !ok {
		return false
	}
	lmaj, lmin, lpat, lpre, ok := ParseVersion(latest)
	if !ok {
		return false
	}
	if lmaj != cmaj {
		return lmaj > cmaj
	}
	if lmin != cmin {
		return lmin > cmin
	}
	if lpat != cpat {
		return lpat > cpat
	}
	// Same major.minor.patch: a pre-release ranks lower than a release.
	if lpre == cpre {
		return false
	}
	if lpre == "" {
		// latest is a release, current is a pre-release: newer.
		return true
	}
	if cpre == "" {
		// latest is a pre-release, current is the release: not newer.
		return false
	}
	// Both are pre-releases of the same version: latest is newer when its
	// identifier sorts after current's.
	return lpre > cpre
}

// PickAsset selects the correct archive for the given OS/arch from release assets.
func PickAsset(assets []Asset, goos, goarch string) (Asset, bool) {
	suffix := fmt.Sprintf("_%s_%s.tar.gz", goos, goarch)
	for _, a := range assets {
		if strings.HasSuffix(a.Name, suffix) {
			return a, true
		}
	}
	return Asset{}, false
}

// CurrentAsset picks the asset for the current runtime OS/arch.
func CurrentAsset(assets []Asset) (Asset, bool) {
	return PickAsset(assets, runtime.GOOS, runtime.GOARCH)
}

const defaultOwner = "yhzion"
const defaultRepo = "portkey"
const githubAPI = "https://api.github.com"

// userAgent is sent on GitHub API requests. GitHub rejects User-Agent-less
// requests with 403, so a non-empty value is required.
const userAgent = "portkey"

// ghRelease is the GitHub API response shape for /releases/latest.
type ghRelease struct {
	TagName string    `json:"tag_name"`
	Assets  []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

// Client checks for updates via GitHub releases.
type Client struct {
	HTTP    *http.Client
	Owner   string
	Repo    string
	BaseURL string
}

// DefaultClient returns a client with sensible defaults.
func DefaultClient() *Client {
	return &Client{
		HTTP:    &http.Client{Timeout: 10 * time.Second},
		Owner:   defaultOwner,
		Repo:    defaultRepo,
		BaseURL: githubAPI,
	}
}

// CheckLatest fetches the latest release from GitHub.
func (c *Client) CheckLatest() (*Release, error) {
	url := fmt.Sprintf(
		"%s/repos/%s/%s/releases/latest",
		c.BaseURL, c.Owner, c.Repo,
	)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("rate limited")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var gh ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&gh); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	assets := make([]Asset, len(gh.Assets))
	for i, a := range gh.Assets {
		assets[i] = Asset(a)
	}

	return &Release{Tag: gh.TagName, Assets: assets}, nil
}
