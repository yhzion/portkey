#!/usr/bin/env bats
# tests/signature.bats — tests for install.sh verify_signature (best-effort matrix)
#
# Stubs minisign on PATH via a temp dir and exercises all three cases:
#   1. minisign present + valid sig  → ok (exit 0)
#   2. minisign present + bad sig    → die (non-zero + error message)
#   3. minisign absent               → warn + continue (exit 0 + warning message)

# Source install.sh functions (non-main) for unit testing.
# main() is defined but not called on source.
setup() {
    source "$(dirname "$BATS_TEST_FILENAME")/../install.sh"

    # Create a scratch directory for stubs and test files.
    TEST_TMPDIR="$(mktemp -d)"

    # Populate dummy checksums.txt and sig file paths used by the tests.
    CHECKSUMS_FILE="${TEST_TMPDIR}/checksums.txt"
    SIG_FILE="${TEST_TMPDIR}/checksums.txt.minisig"

    printf "abc123  portkey_0.1.0_linux_amd64.tar.gz\n" > "$CHECKSUMS_FILE"
    printf "dummy-sig-content\n" > "$SIG_FILE"
}

teardown() {
    rm -rf "$TEST_TMPDIR"
}

# ---------------------------------------------------------------------------
# verify_signature: minisign present and returns success
# ---------------------------------------------------------------------------

@test "verification passes when minisign succeeds" {
    # Stub minisign that always exits 0 (valid signature).
    local stub_dir="${TEST_TMPDIR}/stub_pass"
    mkdir -p "$stub_dir"
    printf '#!/bin/sh\nexit 0\n' > "${stub_dir}/minisign"
    chmod +x "${stub_dir}/minisign"

    local old_path="$PATH"
    export PATH="${stub_dir}:${PATH}"
    run verify_signature "$CHECKSUMS_FILE" "$SIG_FILE"
    export PATH="$old_path"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Signature verified" ]]
}

# ---------------------------------------------------------------------------
# verify_signature: minisign present but returns failure
# ---------------------------------------------------------------------------

@test "install aborts when minisign fails" {
    # Stub minisign that always exits 1 (invalid signature).
    local stub_dir="${TEST_TMPDIR}/stub_fail"
    mkdir -p "$stub_dir"
    printf '#!/bin/sh\nexit 1\n' > "${stub_dir}/minisign"
    chmod +x "${stub_dir}/minisign"

    local old_path="$PATH"
    export PATH="${stub_dir}:${PATH}"
    # die() writes to stderr; bats-core 1.x merges stderr into $output by default.
    run verify_signature "$CHECKSUMS_FILE" "$SIG_FILE"
    export PATH="$old_path"
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Signature verification failed" ]]
}

# ---------------------------------------------------------------------------
# verify_signature: minisign absent — best-effort warn + continue
# ---------------------------------------------------------------------------

@test "best-effort: continues with a warning when minisign is absent" {
    # Use a PATH that has no minisign binary.
    local empty_dir="${TEST_TMPDIR}/stub_empty"
    mkdir -p "$empty_dir"

    local old_path="$PATH"
    export PATH="$empty_dir"
    run verify_signature "$CHECKSUMS_FILE" "$SIG_FILE"
    export PATH="$old_path"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "minisign not found" ]]
}
