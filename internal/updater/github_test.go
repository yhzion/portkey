package updater

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckLatest_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ghRelease{
			TagName: "v0.2.0",
			Assets: []ghAsset{
				{Name: "portkey_0.2.0_linux_arm64.tar.gz", URL: "https://example.com/portkey_0.2.0_linux_arm64.tar.gz"},
				{Name: "checksums.txt", URL: "https://example.com/checksums.txt"},
			},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := &Client{
		HTTP:    srv.Client(),
		Owner:   "yhzion",
		Repo:    "portkey",
		BaseURL: srv.URL,
	}

	rel, err := c.CheckLatest()
	if err != nil {
		t.Fatalf("CheckLatest() error: %v", err)
	}
	if rel.Tag != "v0.2.0" {
		t.Errorf("Tag = %q, want %q", rel.Tag, "v0.2.0")
	}
	if len(rel.Assets) != 2 {
		t.Errorf("len(Assets) = %d, want 2", len(rel.Assets))
	}
}

func TestCheckLatest_RateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	c := &Client{
		HTTP:    srv.Client(),
		Owner:   "yhzion",
		Repo:    "portkey",
		BaseURL: srv.URL,
	}

	_, err := c.CheckLatest()
	if err == nil {
		t.Fatal("expected error for rate limit, got nil")
	}
	if err.Error() != "rate limited" {
		t.Errorf("error = %q, want %q", err.Error(), "rate limited")
	}
}

func TestCheckLatest_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := &Client{
		HTTP:    srv.Client(),
		Owner:   "yhzion",
		Repo:    "portkey",
		BaseURL: srv.URL,
	}

	_, err := c.CheckLatest()
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
}
