#!/usr/bin/env bash
set -euo pipefail

# install.sh — one-line installer for portkey
# Usage: curl -fsSL https://raw.githubusercontent.com/yhzion/portkey/main/install.sh | bash
#        wget -qO- https://raw.githubusercontent.com/yhzion/portkey/main/install.sh | bash

REPO="yhzion/portkey"
BINARY="portkey"

# ── Colors ────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
RESET='\033[0m'

info()  { printf "${CYAN}${BOLD}  ➜${RESET} %s\n" "$*"; }
ok()    { printf "${GREEN}${BOLD}  ✓${RESET} %s\n" "$*"; }
warn()  { printf "${YELLOW}${BOLD}  ⚠${RESET} %s\n" "$*"; }
die()   { printf "${RED}${BOLD}  ✗${RESET} %s\n" "$*" >&2; exit 1; }

# ── Cleanup on exit ───────────────────────────────────────────────────────
TMPDIR=""
cleanup() {
  if [ -n "$TMPDIR" ] && [ -d "$TMPDIR" ]; then
    rm -rf "$TMPDIR"
  fi
}
trap cleanup EXIT

# ── Detect OS ─────────────────────────────────────────────────────────────
detect_os() {
  # Termux runs on Android — Bionic requires its own binary
  if is_termux; then
    echo "android"
    return
  fi

  local os
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  case "$os" in
    darwin) echo "darwin" ;;
    linux)  echo "linux" ;;
    *)      die "Unsupported OS: $(uname -s). Portkey requires macOS or Linux." ;;
  esac
}

# ── Detect architecture ──────────────────────────────────────────────────
detect_arch() {
  local arch
  arch="$(uname -m)"
  case "$arch" in
    x86_64|amd64)      echo "amd64" ;;
    aarch64|arm64)     echo "arm64" ;;
    *)                  die "Unsupported architecture: $arch. Portkey requires amd64 or arm64." ;;
  esac
}

# ── Detect download tool ─────────────────────────────────────────────────
detect_downloader() {
  if command -v curl >/dev/null 2>&1; then
    echo "curl"
  elif command -v wget >/dev/null 2>&1; then
    echo "wget"
  else
    die "Neither curl nor wget found. Please install one and try again."
  fi
}

# ── Download helper ───────────────────────────────────────────────────────
download() {
  local url="$1"
  local dest="$2"

  case "$DOWNLOADER" in
    curl) curl -fsSL "$url" -o "$dest" ;;
    wget) wget -qO "$dest" "$url" ;;
  esac
}

# ── Get latest version from GitHub ────────────────────────────────────────
get_latest_version() {
  local url="https://api.github.com/repos/${REPO}/releases/latest"
  local tmp
  tmp="$(mktemp)"

  download "$url" "$tmp" 2>/dev/null || true

  if [ ! -s "$tmp" ]; then
    rm -f "$tmp"
    die "Failed to fetch latest release from GitHub."
  fi

  # Extract tag_name (e.g. "v0.1.1") — works with grep + sed, no jq needed
  local tag
  tag="$(grep '"tag_name"' "$tmp" | head -1 | sed -E 's/.*"tag_name":\s*"([^"]+)".*/\1/')"
  rm -f "$tmp"

  if [ -z "$tag" ]; then
    die "Could not determine latest version from GitHub response."
  fi

  echo "$tag"
}

# ── Verify checksum ──────────────────────────────────────────────────────
verify_checksum() {
  local tarball="$1"
  local checksums="$2"
  local filename
  filename="$(basename "$tarball")"

  # Extract expected hash for our file
  local expected
  expected="$(grep "  ${filename}$" "$checksums" | awk '{print $1}')"

  if [ -z "$expected" ]; then
    die "Checksum not found for ${filename} in checksums.txt"
  fi

  # Compute actual hash (sha256sum on Linux, shasum -a 256 on macOS)
  local actual
  if command -v sha256sum >/dev/null 2>&1; then
    actual="$(sha256sum "$tarball" | awk '{print $1}')"
  elif command -v shasum >/dev/null 2>&1; then
    actual="$(shasum -a 256 "$tarball" | awk '{print $1}')"
  else
    warn "No sha256 tool found — skipping checksum verification."
    return 0
  fi

  if [ "$actual" != "$expected" ]; then
    die "Checksum mismatch!\n  expected: ${expected}\n  actual:   ${actual}"
  fi

  ok "Checksum verified"
}

# ── Detect Termux ─────────────────────────────────────────────────────────
is_termux() {
  [ -n "${TERMUX_VERSION:-}" ] || [ -x "${PREFIX:-}/bin/termux-setup-storage" ] 2>/dev/null
}

# ── Resolve install directory ─────────────────────────────────────────────
resolve_bindir() {
  # 1. User override
  if [ -n "${PORTKEY_BIN:-}" ]; then
    echo "$PORTKEY_BIN"
    return
  fi

  # 2. Termux: use $PREFIX/bin (already in PATH)
  if is_termux && [ -w "${PREFIX}/bin" ] 2>/dev/null; then
    echo "${PREFIX}/bin"
    return
  fi

  # 3. /usr/local/bin if writable (no sudo needed)
  if [ -w "/usr/local/bin" ] 2>/dev/null; then
    echo "/usr/local/bin"
    return
  fi

  # 4. ~/.local/bin (create if missing)
  local bindir="${HOME}/.local/bin"
  mkdir -p "$bindir"
  echo "$bindir"
}

# ── Detect shell rc file ──────────────────────────────────────────────────
detect_rcfile() {
  local shell_name
  shell_name="$(basename "${SHELL:-}")"

  case "$shell_name" in
    zsh)  echo "${HOME}/.zshrc" ;;
    bash)
      if [ -f "${HOME}/.bashrc" ]; then
        echo "${HOME}/.bashrc"
      elif [ "$(uname -s)" = "Darwin" ] && [ -f "${HOME}/.bash_profile" ]; then
        echo "${HOME}/.bash_profile"
      else
        echo "${HOME}/.bashrc"
      fi
      ;;
    *)    echo "${HOME}/.profile" ;;
  esac
}

# ── Add directory to PATH in shell rc ─────────────────────────────────────
ensure_in_path() {
  local bindir="$1"

  # Check if already in PATH
  case ":${PATH}:" in
    *":${bindir}:"*) NEED_SOURCE=false ;;
    *) NEED_SOURCE=true ;;
  esac

  # Detect shell rc file
  RCFILE="$(detect_rcfile)"

  # If already in PATH, nothing to do
  [ "$NEED_SOURCE" = "false" ] && return 0

  # Check if it's already in the rc file
  if [ -f "$RCFILE" ] && grep -qF "$bindir" "$RCFILE" 2>/dev/null; then
    # Already in rc but not in current session — still need to source
    return 0
  fi

  # Append export line
  {
    echo ""
    echo "# Added by portkey installer"
    echo "export PATH=\"${bindir}:\$PATH\""
  } >> "$RCFILE"
}

# ── Detect Linux distribution ─────────────────────────────────────────────
detect_distro() {
  if is_termux; then
    echo "termux"
    return
  fi

  if [ -f /etc/os-release ]; then
    # shellcheck disable=SC1091
    . /etc/os-release 2>/dev/null || true
    echo "${ID:-unknown}"
    return
  fi

  echo "unknown"
}

# ── SSH install hint per platform/distro ───────────────────────────────────
ssh_install_hint() {
  local distro="${1:-}"
  case "$distro" in
    ubuntu|debian|linuxmint|pop|elementary)
      echo "sudo apt install openssh-client" ;;
    fedora|rhel|centos|rocky|alma)
      echo "sudo dnf install openssh-clients" ;;
    arch|manjaro|endeavouros)
      echo "sudo pacman -S openssh" ;;
    alpine)
      echo "apk add openssh-client" ;;
    termux)
      echo "pkg install openssh" ;;
    darwin)
      echo "brew install openssh" ;;
    *)
      echo "install openssh-client using your package manager" ;;
  esac
}

# ── Check SSH dependency ───────────────────────────────────────────────────
check_ssh_dependency() {
  if command -v ssh >/dev/null 2>&1; then
    ok "OpenSSH client found"
    return 0
  fi

  warn "OpenSSH client not found."
  local distro
  distro="$(detect_distro)"
  local hint
  hint="$(ssh_install_hint "$distro")"
  printf "\n  ${YELLOW}${BOLD}➜ Install it with:${RESET} %s\n\n" "$hint"
  return 0
}

# ── Main ──────────────────────────────────────────────────────────────────
main() {
  printf "\n${BOLD}  Portkey — Installer${RESET}\n\n"

  # Detect platform
  local goos goarch
  goos="$(detect_os)"
  goarch="$(detect_arch)"
  info "Platform: ${goos}/${goarch}"

  # Detect downloader
  local DOWNLOADER
  DOWNLOADER="$(detect_downloader)"
  info "Using: ${DOWNLOADER}"

  # Resolve latest version
  local version
  version="$(get_latest_version)"
  info "Latest: ${version}"

  # Build archive URL
  # goreleaser produces: portkey_{version}_{os}_{arch}.tar.gz
  local version_stripped="${version#v}"
  local filename="${BINARY}_${version_stripped}_${goos}_${goarch}.tar.gz"
  local base_url="https://github.com/${REPO}/releases/download/${version}"
  local tarball_url="${base_url}/${filename}"
  local checksums_url="${base_url}/checksums.txt"

  # Create temp directory
  TMPDIR="$(mktemp -d)"

  # Download checksums
  info "Downloading checksums..."
  download "$checksums_url" "${TMPDIR}/checksums.txt"

  # Download tarball
  info "Downloading ${filename}..."
  download "$tarball_url" "${TMPDIR}/${filename}"

  # Verify checksum
  verify_checksum "${TMPDIR}/${filename}" "${TMPDIR}/checksums.txt"

  # Extract
  info "Extracting..."
  tar xzf "${TMPDIR}/${filename}" -C "$TMPDIR"

  if [ ! -f "${TMPDIR}/${BINARY}" ]; then
    die "Binary '${BINARY}' not found in archive."
  fi

  # Resolve install location
  local bindir
  bindir="$(resolve_bindir)"
  local target="${bindir}/${BINARY}"

  # Install
  info "Installing to ${target}..."
  mv "${TMPDIR}/${BINARY}" "$target"
  chmod +x "$target"

  # Ensure PATH
  ensure_in_path "$bindir"

  # Check SSH dependency
  check_ssh_dependency

  # Done
  printf "\n"
  ok "Portkey ${version} installed successfully!"
  printf "\n  ${BOLD}portkey${RESET} — pick a host and jump in.\n"

  # If bindir is not in PATH, print a prominent notice
  if [ "${NEED_SOURCE:-false}" = "true" ]; then
    printf "\n  ${YELLOW}${BOLD}⚠ Action required:${RESET} ${bindir} is not in your PATH.\n"
    printf "  Run this command to use portkey right away:\n\n"
    printf "    ${BOLD}source %s${RESET}\n\n" "${RCFILE}"
  fi
}

main "$@"
