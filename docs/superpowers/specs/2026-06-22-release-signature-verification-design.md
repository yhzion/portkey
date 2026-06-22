# Release signature verification (minisign) — design

**Issue:** #53, final remaining item ("Integrity theater").
**Date:** 2026-06-22
**Status:** Approved (brainstorming) — pending spec review.

## Problem

`verifyChecksum` (`internal/updater/install.go`) and `install.sh` trust
`checksums.txt` fetched from the same unauthenticated GitHub release. SHA-256
checksums prove **integrity** (no corruption in transit) but not
**authenticity**: a compromised release or MITM can ship a tampered tarball
*plus* a matching `checksums.txt`, and verification passes. This is the
"integrity theater" the review flagged.

## Goal

Establish a cryptographic chain of trust anchored in a public key pinned into
the binary and `install.sh`, so an attacker cannot forge an acceptable release
without the maintainer's private key.

**Trust chain:** pinned public key → `checksums.txt` (minisign signature) →
tarball (SHA-256 listed in `checksums.txt`).

## Decisions (resolved during brainstorming)

| Decision | Choice | Rationale |
|---|---|---|
| Signing scheme | **minisign** (fixed ed25519 keypair) | Go-side verify needs no heavy deps; preserves the existing **local** `release.sh` flow without browser auth; one keypair to manage. |
| Go verifier | **`github.com/jedisct1/go-minisign`** (verify-only, pure Go) | By minisign's author; tiny; correctly handles minisign's trusted-comment + global-signature format. Hand-rolling ed25519 risks format bugs. |
| `install.go` policy | **fail-closed** | If `.minisig` is missing or verification fails, refuse to install with a clear error. All future releases are signed, so this is safe and is the correct default for the in-app auto-updater. |
| `install.sh` policy | **best-effort** | curl\|sh bootstrap users mostly lack `minisign`. If `minisign` is present, verify and `die` on mismatch; if absent, warn and fall back to checksum-only. Avoids breaking `curl\|sh` installs. |
| Release execution | **stays local** (`release.sh` + goreleaser) | Not migrating to GitHub Actions OIDC; fixed-key signing works locally without interactive auth. |

## Architecture

### Release side (signature generation)

- **`.goreleaser.yml`** — add a `signs` block targeting the checksum artifact:
  invoke `minisign` to sign `checksums.txt`, producing `checksums.txt.minisig`,
  and ensure it is uploaded as a release asset alongside `checksums.txt`.
  - Secret key supplied from a file outside the repo (e.g.
    `~/.minisign/portkey.key`); password supplied via the `MINISIGN_PASSWORD`
    environment variable so releases stay non-interactive.
- **`release.sh`** — add `minisign` to `PREREQS`. Document that
  `MINISIGN_PASSWORD` (and the secret key path, if not the default) must be set
  before release.
- **Key generation** — one-time manual step, documented in the runbook
  (AGENTS.md): `minisign -G` to create the keypair; store the secret key
  securely; commit/embed the **public** key only.

### Verification side

- **`internal/updater/pubkey.go`** (new) — single Go source of the pinned key:
  `const MinisignPublicKey = "RW..."` (the minisign public key line, base64).
- **`internal/updater/signature.go`** (new) — `verifySignature(signedData,
  minisig []byte) error` wrapping `go-minisign`: parse the embedded public key,
  parse the `.minisig`, verify the signature over `checksums.txt`. Returns a
  wrapped error on any failure (`%w`).
- **`internal/updater/install.go`** — in `DownloadAndInstall`, after fetching
  `checksums.txt` and before trusting it:
  1. Locate the `checksums.txt.minisig` asset in `rel.Assets`. If absent →
     return an error (fail-closed): "release is not signed; refusing to
     install".
  2. Download it (HTTP status checked, size-capped like other downloads).
  3. `verifySignature(checksumsBytes, minisigBytes)`; on error → abort.
  4. Proceed with the existing tarball checksum verification.
  The signature gates the checksum file; the checksum file gates the tarball.
- **`install.sh`** — embed the same public key in a variable
  (`MINISIGN_PUBKEY="RW..."`). After downloading `checksums.txt`, also download
  `checksums.txt.minisig` (best-effort). Then:
  - If `minisign` is available **and** the `.minisig` was downloaded: run
    `minisign -V -P "$MINISIGN_PUBKEY" -m checksums.txt -x checksums.txt.minisig`;
    `die` on verification failure.
  - If `minisign` is unavailable: `warn` that signature verification was
    skipped (suggest installing minisign) and continue with checksum-only.
  - If `minisign` is available but `.minisig` is missing: `warn` (cannot verify)
    and continue.
  - Then run the existing checksum verification.

## Data flow

```
release.sh → goreleaser build → checksums.txt
           → minisign sign (secret key + MINISIGN_PASSWORD) → checksums.txt.minisig
           → upload both as release assets

install.go (auto-update): pinned pubkey → verify checksums.txt.minisig (FAIL-CLOSED)
           → verify tarball SHA-256 against checksums.txt → atomic replace
install.sh (bootstrap):  pinned pubkey → verify .minisig if minisign present (BEST-EFFORT)
           → verify tarball SHA-256 against checksums.txt → install
```

## Error handling

- All new errors wrap their cause with `%w`; nothing is swallowed (AGENTS.md).
- `install.go` fail-closed: missing-signature and invalid-signature are distinct,
  clearly-worded errors so users understand why an update was refused.
- `install.sh` best-effort: skipped verification is a visible `warn`, never silent.

## Testing

- **Go (`internal/updater/signature_test.go`)** — generate an ephemeral test
  keypair (or commit fixed test fixtures); assert:
  - valid signature over `checksums.txt` passes;
  - tampered `checksums.txt` fails;
  - tampered/garbage `.minisig` fails;
  - wrong-key signature fails.
  Plus an `install.go`-level test that a release with no `.minisig` asset is
  refused (fail-closed). Keep all tests platform-agnostic (no hardcoded
  GOOS/GOARCH; use `runtime.GOOS/GOARCH` if an asset name is needed).
- **bats (`tests/`)** — `install.sh` best-effort matrix: minisign present +
  valid sig (passes), minisign present + invalid sig (dies), minisign absent
  (warns, continues).
- **Key-sync test** — a Go test asserts the public key embedded in `install.sh`
  matches `MinisignPublicKey`, preventing drift between the two copies.

## Components & boundaries

- `pubkey.go` — the single pinned-key constant. Purpose: one source of truth.
- `signature.go` — pure verification logic, no I/O. Testable in isolation with
  in-memory bytes.
- `install.go` — orchestration: fetch assets, call `verifySignature`, then
  checksum, then replace. Depends on `signature.go` + `pubkey.go`.
- `install.sh` — bootstrap mirror of the same chain, best-effort.

## Dependencies

- Add `github.com/jedisct1/go-minisign` (verify-only, pure Go).

## Out of scope

- Existing `install.go` hardening (`maxBinarySize`, `io.LimitReader`, atomic
  replace, tar guards) — unchanged.
- Migrating releases to GitHub Actions / cosign keyless.
- Any unrelated refactor (surgical changes only).
- Retroactively signing past releases (e.g. v0.3.13); verification applies to
  releases published from the first signed version onward.

## Rollout note

The first release after this change must be produced with the signing key in
place (so `checksums.txt.minisig` exists). Because `install.go` is fail-closed,
a release missing the signature asset would be un-installable via the in-app
updater — the release runbook must treat signing as a required step.
