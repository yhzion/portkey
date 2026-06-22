package updater

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"strings"
	"time"
)

// CheckErrorKind classifies why a CheckLatest call failed. The zero value is
// KindUnknown; callers can switch on the kind to render distinct hints.
type CheckErrorKind int

const (
	KindUnknown     CheckErrorKind = iota // unclassified / not set
	KindOffline                           // transport / DNS / connection failure
	KindRateLimited                       // HTTP 429 (Too Many Requests)
	KindNotFound                          // HTTP 404 — no releases published yet
	KindOther                             // HTTP 403 or any other non-200 status
)

// String returns a human-readable name for the kind (used in tests/logging).
func (k CheckErrorKind) String() string {
	switch k {
	case KindOffline:
		return "offline"
	case KindRateLimited:
		return "rate_limited"
	case KindNotFound:
		return "not_found"
	case KindOther:
		return "other"
	default:
		return "unknown"
	}
}

// Sentinel errors returned (wrapped) by CheckLatest. Callers should use
// errors.Is rather than comparing error strings directly; wrapping is safe.
var (
	ErrRateLimited = errors.New("rate limited")
	ErrNoReleases  = errors.New("no releases published yet")
	ErrForbidden   = errors.New("forbidden (check token or repo visibility)")
)

// ClassifyCheckError inspects err returned by CheckLatest and returns the
// appropriate CheckErrorKind. It must not be called with a nil error.
//
// KindOffline covers both genuine connectivity failures and deadline/timeout
// errors: a context.DeadlineExceeded wrapped inside *url.Error satisfies
// net.Error (Timeout() == true), so timed-out checks classify as KindOffline.
func ClassifyCheckError(err error) CheckErrorKind {
	if err == nil {
		return KindUnknown
	}
	// Transport / offline errors contain a wrapped net.Error.
	// This includes context.DeadlineExceeded wrapped by *url.Error.
	var netErr net.Error
	if errors.As(err, &netErr) {
		return KindOffline
	}
	switch {
	case errors.Is(err, ErrRateLimited):
		return KindRateLimited
	case errors.Is(err, ErrNoReleases):
		return KindNotFound
	case errors.Is(err, ErrForbidden):
		return KindOther
	}
	return KindOther
}

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
	// HTTP is used for the short metadata check (CheckLatest). It has a tight
	// timeout (~10 s) so the TUI startup check doesn't stall the user.
	HTTP *http.Client

	// DownloadHTTP is used for binary downloads (DownloadAndInstall). It must
	// NOT share the same tight timeout as HTTP because multi-MB downloads can
	// legitimately take much longer than 10 s on slow connections. A zero
	// Timeout means no fixed deadline; per-request context deadlines may still
	// be applied by the caller.
	DownloadHTTP *http.Client

	Owner   string
	Repo    string
	BaseURL string

	// signPubKey is the minisign public key used to verify checksums.txt.
	// Defaults to MinisignPublicKey; may be overridden in tests.
	signPubKey string
}

// DefaultClient returns a client with sensible defaults.
func DefaultClient() *Client {
	return &Client{
		HTTP:         &http.Client{Timeout: 10 * time.Second},
		DownloadHTTP: &http.Client{Timeout: 5 * time.Minute}, // generous timeout; downloads can take longer
		Owner:        defaultOwner,
		Repo:         defaultRepo,
		BaseURL:      githubAPI,
		signPubKey:   MinisignPublicKey,
	}
}

// CheckLatest fetches the latest release from GitHub.
// The provided ctx is attached to the HTTP request so callers can cancel an
// in-flight check (e.g. when the user quits the TUI). A cancelled or
// deadline-exceeded context causes CheckLatest to return promptly with the
// context error wrapped so that ClassifyCheckError still works.
func (c *Client) CheckLatest(ctx context.Context) (*Release, error) {
	url := fmt.Sprintf(
		"%s/repos/%s/%s/releases/latest",
		c.BaseURL, c.Owner, c.Repo,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// fall through to decode
	case http.StatusTooManyRequests:
		return nil, fmt.Errorf("update check: %w", ErrRateLimited)
	case http.StatusNotFound:
		return nil, fmt.Errorf("update check: %w", ErrNoReleases)
	case http.StatusForbidden:
		return nil, fmt.Errorf("update check: %w", ErrForbidden)
	default:
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
