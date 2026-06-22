package updater

// MinisignPublicKey is the pinned minisign public key (the base64 "RW..." line
// from the keypair's .pub file) used to verify release checksums. The matching
// secret key is held only by the maintainer; see the release runbook in
// AGENTS.md. This same value is embedded in install.sh and kept in sync by
// TestInstallScriptPublicKeyMatches.
const MinisignPublicKey = "RWQ6h3xYCh5o0DVdhljqTXzFkLkXM9k3A2CxwWNr8b9rXYU0qwTeF+TH"
