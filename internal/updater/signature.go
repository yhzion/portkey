package updater

import (
	"fmt"

	minisign "github.com/jedisct1/go-minisign"
)

// verifyMinisign reports whether minisig is a valid minisign signature over
// signedData under pubKey (the base64 "RW..." public-key line). It returns a
// wrapped error on any parse or verification failure.
func verifyMinisign(pubKey string, signedData, minisig []byte) error {
	pk, err := minisign.NewPublicKey(pubKey)
	if err != nil {
		return fmt.Errorf("parse public key: %w", err)
	}
	sig, err := minisign.DecodeSignature(string(minisig))
	if err != nil {
		return fmt.Errorf("parse signature: %w", err)
	}
	ok, err := pk.Verify(signedData, sig)
	if err != nil {
		return fmt.Errorf("verify signature: %w", err)
	}
	if !ok {
		return fmt.Errorf("signature verification failed")
	}
	return nil
}
