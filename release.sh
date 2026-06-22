#!/usr/bin/env bash
set -euo pipefail

# release.sh — calculate version, update changelog, tag, release.
# Usage: ./release.sh [patch|minor|major]

# ── Colors ────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
DIM='\033[2m'
RESET='\033[0m'

info()  { printf "${CYAN}${BOLD}  ➜${RESET} %s\n" "$*"; }
ok()    { printf "${GREEN}${BOLD}  ✓${RESET} %s\n" "$*"; }
warn()  { printf "${YELLOW}${BOLD}  ⚠${RESET} %s\n" "$*"; }
die()   { printf "${RED}${BOLD}  ✗${RESET} %s\n" "$*" >&2; exit 1; }

# ── Prerequisites ────────────────────────────────────────────────────
# Format: "command|install_command|install_hint"
PREREQS=(
  "go||from https://go.dev/dl"
  "git||from https://git-scm.com"
  "svu|go install github.com/caarlos0/svu@latest|go install github.com/caarlos0/svu@latest"
  "goreleaser|go install github.com/goreleaser/goreleaser/v2@latest|go install github.com/goreleaser/goreleaser/v2@latest"
  "minisign||from your package manager: brew install minisign / apt install minisign"
)

printf "\n${BOLD}  Portkey — Prerequisites${RESET}\n\n"

FAILED=()
for entry in "${PREREQS[@]}"; do
  IFS='|' read -r cmd install_cmd install_hint <<< "$entry"
  if command -v "$cmd" &>/dev/null; then
    ok "$cmd"
  elif [ -n "$install_cmd" ]; then
    info "$cmd not found, installing..."
    if $install_cmd 2>/dev/null; then
      ok "$cmd installed"
    else
      warn "Could not install $cmd"
      FAILED+=("$cmd|$install_hint")
    fi
  else
    warn "$cmd not found"
    FAILED+=("$cmd|$install_hint")
  fi
done

if [ ${#FAILED[@]} -gt 0 ]; then
  printf "\n${RED}${BOLD}  ✗${RESET} Missing dependencies. Install manually:\n\n"
  for entry in "${FAILED[@]}"; do
    IFS='|' read -r cmd hint <<< "$entry"
    printf "    ${BOLD}%s${RESET}\n      %s${DIM}%s${RESET}\n" "$cmd" "$ " "$hint"
  done
  printf "\n  Ensure ${BOLD}~/go/bin${RESET} is in your PATH:\n"
  printf "    ${DIM}export PATH=\"\$HOME/go/bin:\$PATH\"${RESET}\n\n"
  exit 1
fi

printf "\n"

if [ -z "${MINISIGN_KEY_FILE:-}" ] || [ -z "${MINISIGN_PASSWORD:-}" ]; then
  die "Set MINISIGN_KEY_FILE (path to secret key) and MINISIGN_PASSWORD before releasing (see AGENTS.md)."
fi

# ── Version ──────────────────────────────────────────────────────────
BUMP="${1:-}"

if [ -n "$BUMP" ]; then
  CURRENT="$(svu current)"
  case "$BUMP" in
    patch) NEXT="$(svu patch)" ;;
    minor) NEXT="$(svu minor)" ;;
    major) NEXT="$(svu major)" ;;
    *) printf "\n  ${BOLD}Usage:${RESET} $0 [patch|minor|major]\n" && exit 1 ;;
  esac
else
  NEXT="$(svu next)"
fi

if [ "$NEXT" = "$(svu current)" ]; then
  die "No version bump detected. Use: $0 [patch|minor|major]"
fi

info "Releasing ${NEXT}\n"

# ── Update CHANGELOG.md ──────────────────────────────────────────────
DATE="$(date +%Y-%m-%d)"
TAG="${NEXT#v}"
HEADER="## [$TAG] - $DATE"

PREV_TAG="$(git describe --tags --abbrev=0 2>/dev/null || echo "")"
if [ -n "$PREV_TAG" ]; then
  LOG="$(git log "$PREV_TAG"..HEAD --pretty=format:"- %s" 2>/dev/null || echo "")"
else
  LOG="$(git log --pretty=format:"- %s" 2>/dev/null || echo "")"
fi

if [ -f CHANGELOG.md ]; then
  TMP="$(mktemp)"
  {
    head -1 CHANGELOG.md
    echo ""
    echo "$HEADER"
    if echo "$LOG" | grep -qi "^- feat"; then
      echo ""
      echo "### Added"
      echo "$LOG" | grep -i "^- feat" | sed 's/^- feat([^)]*): /- /' | sed 's/^- feat: /- /'
    fi
    if echo "$LOG" | grep -qi "^- fix"; then
      echo ""
      echo "### Fixed"
      echo "$LOG" | grep -i "^- fix" | sed 's/^- fix([^)]*): /- /' | sed 's/^- fix: /- /'
    fi
    echo ""
    tail -n +2 CHANGELOG.md
  } > "$TMP"
  mv "$TMP" CHANGELOG.md
else
  echo "# Changelog" > CHANGELOG.md
  echo "" >> CHANGELOG.md
  echo "$HEADER" >> CHANGELOG.md
  echo "" >> CHANGELOG.md
  echo "### Added" >> CHANGELOG.md
  echo "- Initial release: host add, edit, delete, SSH connect" >> CHANGELOG.md
fi

# ── Commit, tag, release ─────────────────────────────────────────────
git add CHANGELOG.md
git commit -m "chore(release): prepare $NEXT"
git tag "$NEXT"

if [ -z "${GITHUB_TOKEN:-}" ] && command -v gh &>/dev/null; then
  GITHUB_TOKEN="$(gh auth token)"
  export GITHUB_TOKEN
fi
goreleaser release --clean
