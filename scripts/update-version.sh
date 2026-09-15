#!/usr/bin/env bash
# Update ProcHub version across all version source files.
# Usage: scripts/update-version.sh <version>   e.g. 0.7.0 or 0.7.0-beta
set -euo pipefail

VERSION="${1:?usage: scripts/update-version.sh <version>}"

# Validate semantic version: X.Y.Z or X.Y.Z-pre (e.g. 0.7.0, 0.7.0-beta.1)
if ! [[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]]; then
  echo "ERROR: invalid version '$VERSION' (expected e.g. 0.7.0 or 0.7.0-beta.1)" >&2
  exit 1
fi

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# Read current version from root package.json (source of truth)
OLD_VERSION="$(perl -ne 'if(/"version":\s*"([^"]+)"/){print $1; exit}' package.json 2>/dev/null || true)"
[ -n "$OLD_VERSION" ] || OLD_VERSION="$(sed -n 's/.*Version:\s*"\([^"]*\)".*/\1/p' app.go | head -1)"

echo ">>> Updating version: ${OLD_VERSION:-unknown} -> $VERSION"

# app.go (Go backend)
if [ -f app.go ]; then
  perl -0pi -e 's/(Version:\s+")[^"]*(")/${1}'"$VERSION"'${2}/' app.go
  echo "  updated app.go"
fi

# package.json (root + frontend, single top-level version field)
for f in package.json frontend/package.json; do
  if [ -f "$f" ]; then
    perl -0pi -e 's/("version":\s*")[^"]*(")/${1}'"$VERSION"'${2}/' "$f"
    echo "  updated $f"
  fi
done

# package-lock.json: replace only the exact old version string
# (avoids touching dependency version fields like "5.2.0")
if [ -n "$OLD_VERSION" ]; then
  for f in package-lock.json frontend/package-lock.json; do
    if [ -f "$f" ]; then
      perl -0pi -e 's/\Q'"$OLD_VERSION"'\E/'"$VERSION"'/g' "$f"
      echo "  updated $f"
    fi
  done
fi

# tests/screenshot.ts (Pro-only mock, may not exist in Open version)
if [ -f tests/screenshot.ts ] && [ -n "$OLD_VERSION" ]; then
  perl -0pi -e 's/v\Q'"$OLD_VERSION"'\E/v'"$VERSION"'/g' tests/screenshot.ts
  echo "  updated tests/screenshot.ts"
fi

echo ">>> Done. Version is now $VERSION"
