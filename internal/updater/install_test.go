package updater

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestVerifyChecksumNon200 verifies that a non-200 response when fetching
// checksums.txt is surfaced as an error mentioning the status rather than
// silently succeeding (MITM / outage hardening).
func TestVerifyChecksumNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := &Client{HTTP: srv.Client()}
	assets := []Asset{{Name: "checksums.txt", URL: srv.URL}}

	err := c.verifyChecksum(assets, "portkey_0.1.0_linux_amd64.tar.gz", []byte("data"))
	if err == nil {
		t.Fatal("verifyChecksum() error = nil, want error for non-200 checksums response")
	}
	if !strings.Contains(err.Error(), "status") {
		t.Errorf("error = %q, want it to mention the status", err.Error())
	}
}
