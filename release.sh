#!/usr/bin/env bash
set -euo pipefail

# release.sh — calculate version, update changelog, tag, release.
# Usage: ./release.sh [patch|minor|major]

BUMP="${1:-}"

if ! command -v svu &>/dev/null; then
  echo "svu not found. Install: go install github.com/caarlos0/svu@latest"
  exit 1
fi

if [ -n "$BUMP" ]; then
  CURRENT="$(svu current)"
  case "$BUMP" in
    patch) NEXT="$(svu patch)" ;;
    minor) NEXT="$(svu minor)" ;;
    major) NEXT="$(svu major)" ;;
    *) echo "Usage: $0 [patch|minor|major]" && exit 1 ;;
  esac
else
  NEXT="$(svu next)"
fi

if [ "$NEXT" = "$(svu current)" ]; then
  echo "No version bump detected. Use: $0 [patch|minor|major]"
  exit 1
fi

echo "Releasing $NEXT"

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

if command -v goreleaser &>/dev/null; then
  if [ -z "${GITHUB_TOKEN:-}" ] && command -v gh &>/dev/null; then
    GITHUB_TOKEN="$(gh auth token)"
    export GITHUB_TOKEN
  fi
  goreleaser release --clean
else
  echo "goreleaser not found. Install: go install github.com/goreleaser/goreleaser/v2@latest"
  echo "Tag $NEXT created. Run 'goreleaser release --clean' manually."
fi
