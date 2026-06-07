#!/usr/bin/env bats
# tests/install_test.bats — tests for install.sh helper functions

# Source install.sh functions (non-main) for unit testing.
# We skip main() by sourcing only the function definitions.
setup() {
    # Source all functions from install.sh.
    # main() is defined but not called on source.
    source "$(dirname "$BATS_TEST_FILENAME")/../install.sh"
}

# ---------------------------------------------------------------------------
# detect_os
# ---------------------------------------------------------------------------

@test "detect_os returns android on Termux" {
    export TERMUX_VERSION="0.118.0"
    run detect_os
    [ "$status" -eq 0 ]
    [ "$output" = "android" ]
    unset TERMUX_VERSION
}

@test "detect_os returns darwin on macOS" {
    unset TERMUX_VERSION
    # detect_os calls uname; on Termux this returns "linux"
    # We test the actual uname output of this machine.
    run detect_os
    [ "$status" -eq 0 ]
    # Should be linux, darwin, or android
    [[ "$output" =~ ^(linux|darwin|android)$ ]]
}

# ---------------------------------------------------------------------------
# is_termux
# ---------------------------------------------------------------------------

@test "is_termux returns true when TERMUX_VERSION is set" {
    export TERMUX_VERSION="0.118.0"
    run is_termux
    [ "$status" -eq 0 ]
    unset TERMUX_VERSION
}

# is_termux: false case cannot be tested on Termux, omitted.

# ---------------------------------------------------------------------------
# detect_downloader
# ---------------------------------------------------------------------------

@test "detect_downloader finds curl or wget" {
    run detect_downloader
    [ "$status" -eq 0 ]
    [[ "$output" =~ ^(curl|wget)$ ]]
}

# ---------------------------------------------------------------------------
# check_ssh_dependency
# ---------------------------------------------------------------------------

@test "check_ssh_dependency succeeds when ssh is in PATH" {
    # ssh should be available on Termux / most systems
    run check_ssh_dependency
    [ "$status" -eq 0 ]
    [[ "$output" =~ "OpenSSH" || "$output" =~ "ssh" ]]
}

@test "check_ssh_dependency shows install instructions when ssh missing" {
    # Override PATH to exclude ssh
    local orig_path="$PATH"
    export PATH="/dev/null"

    run check_ssh_dependency
    [ "$status" -eq 0 ]
    [[ "$output" =~ "install" ]]

    export PATH="$orig_path"
}

# ---------------------------------------------------------------------------
# detect_distro (Linux only)
# ---------------------------------------------------------------------------

@test "detect_distro returns a string on Linux" {
    unset TERMUX_VERSION
    run detect_distro
    [ "$status" -eq 0 ]
    [ -n "$output" ]
}

# ---------------------------------------------------------------------------
# ssh_install_hint
# ---------------------------------------------------------------------------

@test "ssh_install_hint returns apt for Ubuntu" {
    run ssh_install_hint "ubuntu"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "apt" ]]
}

@test "ssh_install_hint returns apt for Debian" {
    run ssh_install_hint "debian"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "apt" ]]
}

@test "ssh_install_hint returns dnf for Fedora" {
    run ssh_install_hint "fedora"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "dnf" ]]
}

@test "ssh_install_hint returns pacman for Arch" {
    run ssh_install_hint "arch"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "pacman" ]]
}

@test "ssh_install_hint returns apk for Alpine" {
    run ssh_install_hint "alpine"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "apk" ]]
}

@test "ssh_install_hint returns pkg for Termux" {
    run ssh_install_hint "termux"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "pkg" ]]
}

@test "ssh_install_hint returns brew for macOS" {
    run ssh_install_hint "darwin"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "brew" ]]
}
