package updater

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"testing"
)

// makeMinisignFixture builds an ephemeral keypair and a valid minisign
// signature over message, returning the public-key line and the .minisig file
// contents. Layout matches the minisign legacy "Ed" format that go-minisign
// verifies: pubkey = base64(algo[2] || keyID[8] || ed25519Pub[32]); message
// signature line = base64(algo[2] || keyID[8] || ed25519Sig[64]); global
// signature = ed25519 over (messageSig || trustedComment).
func makeMinisignFixture(tb testing.TB, message []byte, trustedComment string) (pubKeyLine, minisig string) {
	tb.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		tb.Fatalf("genkey: %v", err)
	}
	var keyID [8]byte
	if _, err := rand.Read(keyID[:]); err != nil {
		tb.Fatalf("keyid: %v", err)
	}
	algo := []byte{'E', 'd'}
	msgSig := ed25519.Sign(priv, message)
	global := ed25519.Sign(priv, append(append([]byte{}, msgSig...), []byte(trustedComment)...))

	pubBlob := append(append(append([]byte{}, algo...), keyID[:]...), pub...)
	sigBlob := append(append(append([]byte{}, algo...), keyID[:]...), msgSig...)

	pubKeyLine = base64.StdEncoding.EncodeToString(pubBlob)
	minisig = fmt.Sprintf(
		"untrusted comment: test\n%s\ntrusted comment: %s\n%s\n",
		base64.StdEncoding.EncodeToString(sigBlob),
		trustedComment,
		base64.StdEncoding.EncodeToString(global),
	)
	return pubKeyLine, minisig
}

func TestVerifyMinisign_Valid(t *testing.T) {
	msg := []byte("checksums-file-contents\n")
	pubLine, sig := makeMinisignFixture(t, msg, "portkey test")
	if err := verifyMinisign(pubLine, msg, []byte(sig)); err != nil {
		t.Fatalf("valid signature rejected: %v", err)
	}
}

func TestVerifyMinisign_TamperedData(t *testing.T) {
	msg := []byte("checksums-file-contents\n")
	pubLine, sig := makeMinisignFixture(t, msg, "portkey test")
	if err := verifyMinisign(pubLine, []byte("tampered\n"), []byte(sig)); err == nil {
		t.Fatal("tampered data accepted")
	}
}

func TestVerifyMinisign_WrongKey(t *testing.T) {
	msg := []byte("data\n")
	_, sig := makeMinisignFixture(t, msg, "portkey test")
	otherPub, _ := makeMinisignFixture(t, []byte("x"), "x")
	if err := verifyMinisign(otherPub, msg, []byte(sig)); err == nil {
		t.Fatal("signature verified under wrong key")
	}
}

func TestVerifyMinisign_GarbageSignature(t *testing.T) {
	pubLine, _ := makeMinisignFixture(t, []byte("data"), "c")
	if err := verifyMinisign(pubLine, []byte("data"), []byte("not a minisig")); err == nil {
		t.Fatal("garbage signature accepted")
	}
}
