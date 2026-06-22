package updater

import (
	"os"
	"regexp"
	"testing"
)

// TestInstallScriptPublicKeyMatches ensures install.sh pins the SAME minisign
// public key as MinisignPublicKey, preventing drift between the two copies.
func TestInstallScriptPublicKeyMatches(t *testing.T) {
	data, err := os.ReadFile("../../install.sh")
	if err != nil {
		t.Fatalf("read install.sh: %v", err)
	}
	re := regexp.MustCompile(`MINISIGN_PUBKEY="([^"]+)"`)
	m := re.FindSubmatch(data)
	if m == nil {
		t.Fatal("MINISIGN_PUBKEY not found in install.sh")
	}
	if got := string(m[1]); got != MinisignPublicKey {
		t.Errorf("install.sh key %q != MinisignPublicKey %q", got, MinisignPublicKey)
	}
}
