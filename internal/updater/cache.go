package updater

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// UpdateCheckTTL is the minimum interval between live GitHub update-check
// requests when using the CachingChecker. One request per day keeps the TUI
// well inside GitHub's unauthenticated API rate limit (60 req/hr/IP).
const UpdateCheckTTL = 24 * time.Hour

// releaseChecker is the minimal interface that CachingChecker wraps.
// *updater.Client satisfies it, as does any fake used in tests.
type releaseChecker interface {
	CheckLatest(ctx context.Context) (*Release, error)
}

// cacheEntry is the JSON shape persisted to disk.
type cacheEntry struct {
	Release   *Release  `json:"release"`
	CheckedAt time.Time `json:"checked_at"`
}

// CachingChecker wraps a releaseChecker and serves cached results within a
// configurable TTL. It satisfies the tui.UpdateChecker interface so it can be
// passed directly to tui.InitialModel without any other changes.
//
// Cache reads and writes are best-effort: any IO or JSON error is silently
// treated as "no cache" and falls through to a live inner check. This
// intentional swallow is documented here so it does not read as a violation of
// the project's "never swallow errors" rule — the cache is a performance
// optimisation and a failure to read/write it must never block the user or
// surface a startup error.
type CachingChecker struct {
	inner     releaseChecker
	cachePath string
	ttl       time.Duration
}

// NewCachingChecker returns a CachingChecker that wraps inner, stores the
// result at cachePath, and skips the live check for ttl after the last
// successful fetch. Inject the cache path from main.go (built from
// config.ConfigDir()) to keep the updater package free of a config import.
func NewCachingChecker(inner releaseChecker, cachePath string, ttl time.Duration) *CachingChecker {
	return &CachingChecker{inner: inner, cachePath: cachePath, ttl: ttl}
}

// CheckLatest satisfies tui.UpdateChecker. It returns a cached *Release when
// the cache is fresh (age < ttl); otherwise it calls inner, updates the cache
// on success, and returns the result.
func (c *CachingChecker) CheckLatest(ctx context.Context) (*Release, error) {
	// Try to serve from cache. Any read or parse error is intentionally ignored
	// (best-effort cache — see type-level comment) and falls through to a live check.
	if rel := c.loadFromCache(); rel != nil {
		return rel, nil
	}

	rel, err := c.inner.CheckLatest(ctx)
	if err != nil {
		return nil, err
	}

	// Persist the fresh result. Write errors are intentionally ignored
	// (best-effort cache — see type-level comment); the caller still gets the
	// live result regardless.
	c.writeToCache(rel)

	return rel, nil
}

// loadFromCache reads the cache file and returns a *Release if it exists and
// is still within the TTL. Returns nil on any error or expiry.
func (c *CachingChecker) loadFromCache() *Release {
	data, err := os.ReadFile(c.cachePath)
	if err != nil {
		// File not found or unreadable — treat as no cache.
		return nil //nolint:nilerr // best-effort cache IO (see type-level comment)
	}

	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		// Corrupt JSON — treat as no cache.
		return nil //nolint:nilerr // best-effort cache IO (see type-level comment)
	}

	if entry.Release == nil || time.Since(entry.CheckedAt) >= c.ttl {
		return nil
	}

	return entry.Release
}

// writeToCache persists rel and the current timestamp to the cache file.
// Any write error is silently discarded (best-effort — see type-level comment).
func (c *CachingChecker) writeToCache(rel *Release) {
	entry := cacheEntry{Release: rel, CheckedAt: time.Now()}
	data, err := json.Marshal(entry)
	if err != nil {
		return // best-effort cache IO (see type-level comment)
	}
	// Ensure the parent directory exists (e.g. on a fresh install before any
	// other code path has created <config dir>/portkey).
	if err := os.MkdirAll(filepath.Dir(c.cachePath), 0o755); err != nil {
		return // best-effort cache IO (see type-level comment)
	}
	// Write with permissions 0600 so only the owner can read the cache file.
	_ = os.WriteFile(c.cachePath, data, 0o600) // best-effort cache IO (see type-level comment)
}
