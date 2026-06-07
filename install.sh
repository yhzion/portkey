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

# ── Resolve install directory ─────────────────────────────────────────────
resolve_bindir() {
  # 1. User override
  if [ -n "${PORTKEY_BIN:-}" ]; then
    echo "$PORTKEY_BIN"
    return
  fi

  # 2. /usr/local/bin if writable (no sudo needed)
  if [ -w "/usr/local/bin" ] 2>/dev/null; then
    echo "/usr/local/bin"
    return
  fi

  # 3. ~/.local/bin (create if missing)
  local bindir="${HOME}/.local/bin"
  mkdir -p "$bindir"
  echo "$bindir"
}

# ── Add directory to PATH in shell rc ─────────────────────────────────────
ensure_in_path() {
  local bindir="$1"

  # Check if already in PATH
  case ":${PATH}:" in
    *":${bindir}:"*) return 0 ;;
  esac

  # Detect shell rc file
  local rcfile=""
  local shell_name
  shell_name="$(basename "${SHELL:-}")"

  case "$shell_name" in
    zsh)  rcfile="${HOME}/.zshrc" ;;
    bash)
      # Prefer .bashrc for interactive, .bash_profile on macOS
      if [ -f "${HOME}/.bashrc" ]; then
        rcfile="${HOME}/.bashrc"
      elif [ "$(uname -s)" = "Darwin" ] && [ -f "${HOME}/.bash_profile" ]; then
        rcfile="${HOME}/.bash_profile"
      else
        rcfile="${HOME}/.bashrc"
      fi
      ;;
    *)    rcfile="${HOME}/.profile" ;;
  esac

  # Check if it's already in the rc file
  if [ -f "$rcfile" ] && grep -qF "$bindir" "$rcfile" 2>/dev/null; then
    return 0
  fi

  # Append export line
  {
    echo ""
    echo "# Added by portkey installer"
    echo "export PATH=\"${bindir}:\$PATH\""
  } >> "$rcfile"

  warn "${bindir} is not in your PATH."
  echo ""
  echo "  Added this line to ${rcfile}:"
  echo "    export PATH=\"${bindir}:\$PATH\""
  echo ""
  echo "  Run this to update your current session:"
  echo "    source ${rcfile}"
  echo ""
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

  # Done
  printf "\n"
  ok "Portkey ${version} installed successfully!"
  printf "\n  ${BOLD}portkey${RESET} — pick a host and jump in.\n\n"
}

main "$@"
