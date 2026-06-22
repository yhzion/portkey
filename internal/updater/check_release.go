package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// tagRe is the allowlist for a version-ish tag. It deliberately excludes
// slashes, whitespace, control characters, and double-dots to prevent
// path-traversal / injection into the GitHub API URL path.
var tagRe = regexp.MustCompile(`^v?[0-9A-Za-z.\-+]+$`)

// ValidateTag returns an error if tag is empty, contains slashes, whitespace,
// "..", or does not match the conservative version allowlist. It is exported
// so callers can validate a tag before passing it to CheckRelease.
func ValidateTag(tag string) error {
	if tag == "" {
		return fmt.Errorf("tag must not be empty")
	}
	if strings.Contains(tag, "/") {
		return fmt.Errorf("invalid tag %q: contains '/'", tag)
	}
	if strings.Contains(tag, "..") {
		return fmt.Errorf("invalid tag %q: contains '..'", tag)
	}
	// Reject any whitespace or control characters.
	for _, r := range tag {
		if r <= 0x20 || r == 0x7f {
			return fmt.Errorf("invalid tag %q: contains whitespace or control character", tag)
		}
	}
	if !tagRe.MatchString(tag) {
		return fmt.Errorf("invalid tag %q: does not match version pattern", tag)
	}
	return nil
}

// CheckRelease fetches a specific release by tag from GitHub.
// It validates tag before interpolating it into the URL (security: prevents
// path-traversal / injection). It mirrors CheckLatest's request style and
// error classification.
func (c *Client) CheckRelease(ctx context.Context, tag string) (*Release, error) {
	if err := ValidateTag(tag); err != nil {
		return nil, err
	}

	url := fmt.Sprintf(
		"%s/repos/%s/%s/releases/tags/%s",
		c.BaseURL, c.Owner, c.Repo, tag,
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
		return nil, fmt.Errorf("fetch release %s: %w", tag, err)
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
