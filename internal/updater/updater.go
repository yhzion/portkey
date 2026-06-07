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

// ParseVersion parses "vX.Y.Z" or "X.Y.Z" into major, minor, patch.
func ParseVersion(tag string) (int, int, int, bool) {
	tag = strings.TrimPrefix(tag, "v")
	parts := strings.SplitN(tag, ".", 3)
	if len(parts) != 3 {
		return 0, 0, 0, false
	}
	var maj, min, pat int
	if _, err := fmt.Sscanf(parts[0], "%d", &maj); err != nil {
		return 0, 0, 0, false
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &min); err != nil {
		return 0, 0, 0, false
	}
	if _, err := fmt.Sscanf(parts[2], "%d", &pat); err != nil {
		return 0, 0, 0, false
	}
	return maj, min, pat, true
}

// IsNewer returns true if latest > current (semver comparison).
// Returns false if either version is unparseable.
func IsNewer(current, latest string) bool {
	cmaj, cmin, cpat, ok := ParseVersion(current)
	if !ok {
		return false
	}
	lmaj, lmin, lpat, ok := ParseVersion(latest)
	if !ok {
		return false
	}
	if lmaj != cmaj {
		return lmaj > cmaj
	}
	if lmin != cmin {
		return lmin > cmin
	}
	return lpat > cpat
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
	resp, err := c.HTTP.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
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
		assets[i] = Asset{Name: a.Name, URL: a.URL}
	}

	return &Release{Tag: gh.TagName, Assets: assets}, nil
}
