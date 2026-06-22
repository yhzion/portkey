package updater

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestCheckLatest_ContextCancel verifies that a cancelled context aborts an
// in-flight CheckLatest call promptly and returns a context error.
func TestCheckLatest_ContextCancel(t *testing.T) {
	// Server that blocks indefinitely (never responds).
	ready := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(ready)
		// Block until the client disconnects.
		<-r.Context().Done()
	}))
	defer srv.Close()

	c := &Client{HTTP: srv.Client(), Owner: "o", Repo: "r", BaseURL: srv.URL}
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		_, err := c.CheckLatest(ctx)
		done <- err
	}()

	// Wait until the server received the request, then cancel.
	<-ready
	cancel()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("CheckLatest() returned nil after context cancel, want error")
		}
		if !errors.Is(err, context.Canceled) {
			t.Errorf("CheckLatest() error = %v, want to wrap context.Canceled", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("CheckLatest() did not return promptly after context cancel")
	}
}

// TestCheckLatest_DeadlineExceeded_ClassifiesAsOffline verifies that a
// deadline-exceeded error still classifies as KindOffline (preserves Task 1
// behaviour — context.DeadlineExceeded wrapped in *url.Error satisfies
// net.Error).
func TestCheckLatest_DeadlineExceeded_ClassifiesAsOffline(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	c := &Client{HTTP: srv.Client(), Owner: "o", Repo: "r", BaseURL: srv.URL}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := c.CheckLatest(ctx)
	if err == nil {
		t.Fatal("expected error from timed-out check, got nil")
	}
	kind := ClassifyCheckError(err)
	if kind != KindOffline {
		t.Errorf("ClassifyCheckError(deadline exceeded) = %v, want KindOffline", kind)
	}
}

// TestDownloadClient_TimeoutDecoupledFromCheck verifies that the download path
// uses a different (longer/no fixed timeout) HTTP client than the 10s check
// client.  We assert via the seam: DefaultClient() should have a check-client
// Timeout of ≤10s, and DownloadHTTP should have a longer or zero Timeout so
// multi-MB downloads are not cut off by the check deadline.
func TestDownloadClient_TimeoutDecoupledFromCheck(t *testing.T) {
	c := DefaultClient()

	// Check client must be bounded (≤ 10s) to keep checks fast.
	if c.HTTP.Timeout == 0 || c.HTTP.Timeout > 10*time.Second {
		t.Errorf("check HTTP client Timeout = %v, want > 0 and ≤ 10s", c.HTTP.Timeout)
	}

	// Download client must NOT share the same tight timeout so large binaries
	// can complete. A zero Timeout means the download relies on request-level
	// context (acceptable); any value > 10s is also fine.
	if c.DownloadHTTP.Timeout != 0 && c.DownloadHTTP.Timeout <= 10*time.Second {
		t.Errorf(
			"download HTTP client Timeout = %v, want 0 (unlimited) or > 10s",
			c.DownloadHTTP.Timeout,
		)
	}
}
