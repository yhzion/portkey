package updater

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// fakeChecker is a call-counting fake inner checker for cache tests.
type fakeChecker struct {
	calls  int
	result *Release
	err    error
}

func (f *fakeChecker) CheckLatest(_ context.Context) (*Release, error) {
	f.calls++
	return f.result, f.err
}

// writeCacheFile writes a cache entry directly to disk with the given checkedAt
// time, allowing tests to control whether the cache is fresh or stale without
// real sleeps.
func writeCacheFile(t *testing.T, path string, rel *Release, checkedAt time.Time) {
	t.Helper()
	entry := cacheEntry{Release: rel, CheckedAt: checkedAt}
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal cache: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write cache file: %v", err)
	}
}

func TestCachingChecker_FreshCache_CallsInnerOnce(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "update-check.json")

	rel := &Release{Tag: "v1.2.3", Assets: []Asset{{Name: "a.tar.gz", URL: "https://example.com/a.tar.gz"}}}
	fake := &fakeChecker{result: rel}

	cc := NewCachingChecker(fake, cachePath, UpdateCheckTTL)

	// First call: no cache exists → must call inner.
	got, err := cc.CheckLatest(context.Background())
	if err != nil {
		t.Fatalf("first CheckLatest() error = %v", err)
	}
	if got.Tag != "v1.2.3" {
		t.Errorf("Tag = %q, want %q", got.Tag, "v1.2.3")
	}
	if fake.calls != 1 {
		t.Errorf("inner called %d times, want 1", fake.calls)
	}

	// Cache file must have been written.
	if _, err := os.Stat(cachePath); err != nil {
		t.Errorf("cache file not written: %v", err)
	}
}

func TestCachingChecker_WithinTTL_DoesNotCallInner(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "update-check.json")

	rel := &Release{Tag: "v1.2.3", Assets: []Asset{{Name: "a.tar.gz", URL: "https://example.com/a.tar.gz"}}}

	// Seed a fresh cache (checkedAt = now).
	writeCacheFile(t, cachePath, rel, time.Now())

	fake := &fakeChecker{result: rel}
	cc := NewCachingChecker(fake, cachePath, UpdateCheckTTL)

	got, err := cc.CheckLatest(context.Background())
	if err != nil {
		t.Fatalf("CheckLatest() error = %v", err)
	}
	if got.Tag != "v1.2.3" {
		t.Errorf("Tag = %q, want %q", got.Tag, "v1.2.3")
	}
	if fake.calls != 0 {
		t.Errorf("inner called %d times within TTL, want 0", fake.calls)
	}
}

func TestCachingChecker_PastTTL_CallsInnerAgain(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "update-check.json")

	oldRel := &Release{Tag: "v1.0.0"}
	newRel := &Release{Tag: "v1.2.3", Assets: []Asset{{Name: "a.tar.gz", URL: "https://example.com/a.tar.gz"}}}

	// Seed a stale cache (checkedAt = 25 hours ago, well past 24h TTL).
	writeCacheFile(t, cachePath, oldRel, time.Now().Add(-25*time.Hour))

	fake := &fakeChecker{result: newRel}
	cc := NewCachingChecker(fake, cachePath, UpdateCheckTTL)

	got, err := cc.CheckLatest(context.Background())
	if err != nil {
		t.Fatalf("CheckLatest() error = %v", err)
	}
	if fake.calls != 1 {
		t.Errorf("inner called %d times past TTL, want 1", fake.calls)
	}
	if got.Tag != "v1.2.3" {
		t.Errorf("Tag = %q, want %q", got.Tag, "v1.2.3")
	}
}

func TestCachingChecker_InnerError_NotCached(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "update-check.json")

	innerErr := errors.New("network unreachable")
	fake := &fakeChecker{err: innerErr}
	cc := NewCachingChecker(fake, cachePath, UpdateCheckTTL)

	_, err := cc.CheckLatest(context.Background())
	if err == nil {
		t.Fatal("expected error from inner, got nil")
	}
	if !errors.Is(err, innerErr) {
		t.Errorf("error = %v, want to wrap %v", err, innerErr)
	}

	// Cache file must NOT be written on inner error.
	if _, statErr := os.Stat(cachePath); !os.IsNotExist(statErr) {
		t.Errorf("cache file should not exist after inner error, stat = %v", statErr)
	}
	if fake.calls != 1 {
		t.Errorf("inner called %d times, want 1", fake.calls)
	}
}

func TestCachingChecker_CorruptCache_FallsBackToLive(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "update-check.json")

	// Write corrupt JSON to the cache file.
	if err := os.WriteFile(cachePath, []byte("{not valid json}"), 0o600); err != nil {
		t.Fatalf("write corrupt cache: %v", err)
	}

	rel := &Release{Tag: "v2.0.0"}
	fake := &fakeChecker{result: rel}
	cc := NewCachingChecker(fake, cachePath, UpdateCheckTTL)

	// Corrupt cache must not surface an error; must fall back to live check.
	got, err := cc.CheckLatest(context.Background())
	if err != nil {
		t.Fatalf("CheckLatest() error = %v (want no error on corrupt cache)", err)
	}
	if fake.calls != 1 {
		t.Errorf("inner called %d times on corrupt cache, want 1", fake.calls)
	}
	if got.Tag != "v2.0.0" {
		t.Errorf("Tag = %q, want %q", got.Tag, "v2.0.0")
	}
}

func TestCachingChecker_CacheHit_FullReleaseData(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "update-check.json")

	rel := &Release{
		Tag: "v3.1.4",
		Assets: []Asset{
			{Name: "portkey_3.1.4_linux_amd64.tar.gz", URL: "https://example.com/portkey_3.1.4_linux_amd64.tar.gz"},
			{Name: "checksums.txt", URL: "https://example.com/checksums.txt"},
		},
	}

	// Seed a fresh cache.
	writeCacheFile(t, cachePath, rel, time.Now())

	fake := &fakeChecker{} // inner never called on a fresh cache hit
	cc := NewCachingChecker(fake, cachePath, UpdateCheckTTL)

	got, err := cc.CheckLatest(context.Background())
	if err != nil {
		t.Fatalf("CheckLatest() error = %v", err)
	}
	if got.Tag != rel.Tag {
		t.Errorf("Tag = %q, want %q", got.Tag, rel.Tag)
	}
	if len(got.Assets) != len(rel.Assets) {
		t.Errorf("len(Assets) = %d, want %d", len(got.Assets), len(rel.Assets))
	}
	for i, a := range rel.Assets {
		if got.Assets[i].Name != a.Name {
			t.Errorf("Assets[%d].Name = %q, want %q", i, got.Assets[i].Name, a.Name)
		}
		if got.Assets[i].URL != a.URL {
			t.Errorf("Assets[%d].URL = %q, want %q", i, got.Assets[i].URL, a.URL)
		}
	}
	if fake.calls != 0 {
		t.Errorf("inner called %d times on fresh cache hit, want 0", fake.calls)
	}
}
