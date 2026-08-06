#!/usr/bin/env bash
# Package ProcHub Linux release: deb + AppImage
#
# Usage: scripts/package-linux.sh <goarch> <version>
#   goarch:  amd64 | arm64
#   version: e.g. 0.6.0
#
# Requires: wails build output at build/bin/ProcHub, build/appicon.png
# Outputs:  ProcHub-<version>-linux-<goarch>.deb / .AppImage (repo root)

set -euo pipefail

GOARCH="${1:?goarch required (amd64|arm64)}"
VERSION="${2:?version required}"

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

case "$GOARCH" in
  amd64)
    DEB_ARCH="amd64"
    APPIMAGE_TOOL="appimagetool-x86_64.AppImage"
    ;;
  arm64)
    DEB_ARCH="arm64"
    APPIMAGE_TOOL="appimagetool-aarch64.AppImage"
    ;;
  *)
    echo "ERROR: unsupported goarch: $GOARCH" >&2
    exit 1
    ;;
esac

DESKTOP_CONTENT='[Desktop Entry]
Name=ProcHub
Comment=Manage processes easily
Exec=prochub
Icon=prochub
Terminal=false
Type=Application
Categories=Utility;System;
'

# ── deb (via fpm) ──────────────────────────────────────────────
if ! command -v fpm >/dev/null 2>&1; then
  echo "Installing fpm..."
  sudo apt-get install -y ruby ruby-dev rubygems >/dev/null 2>&1
  sudo gem install fpm >/dev/null
fi

PKG_DIR="pkg-deb"
rm -rf "$PKG_DIR"
mkdir -p "$PKG_DIR/usr/bin" "$PKG_DIR/usr/share/applications" "$PKG_DIR/usr/share/icons/hicolor/256x256/apps"
cp build/bin/ProcHub "$PKG_DIR/usr/bin/prochub"
printf '%s' "$DESKTOP_CONTENT" > "$PKG_DIR/usr/share/applications/prochub.desktop"
cp build/appicon.png "$PKG_DIR/usr/share/icons/hicolor/256x256/apps/prochub.png"

fpm -s dir -t deb \
  -n prochub \
  -v "$VERSION" \
  -a "$DEB_ARCH" \
  --description "ProcHub - Manage processes easily" \
  --category "Utility" \
  --maintainer "MZ <modstart@163.com>" \
  --url "https://modstart.com" \
  --license "Apache-2.0" \
  -C "$PKG_DIR" \
  -p "ProcHub-${VERSION}-linux-${GOARCH}.deb" >/dev/null
rm -rf "$PKG_DIR"
echo "✅ deb: ProcHub-${VERSION}-linux-${GOARCH}.deb"

# ── AppImage (via appimagetool) ────────────────────────────────
if [ ! -f appimagetool ]; then
  echo "Downloading appimagetool..."
  wget -q "https://github.com/AppImage/appimagetool/releases/download/continuous/${APPIMAGE_TOOL}" -O appimagetool
  chmod +x appimagetool
fi

APPDIR="AppDir"
rm -rf "$APPDIR"
mkdir -p "$APPDIR/usr/bin"
cp build/bin/ProcHub "$APPDIR/usr/bin/prochub"
cp build/appicon.png "$APPDIR/prochub.png"
printf '%s' "$DESKTOP_CONTENT" > "$APPDIR/prochub.desktop"
printf '#!/bin/sh\nexec "$(dirname "$0")/usr/bin/prochub" "$@"\n' > "$APPDIR/AppRun"
chmod +x "$APPDIR/AppRun"

APPIMAGE_EXTRACT_AND_RUN=1 ./appimagetool "$APPDIR" "ProcHub-${VERSION}-linux-${GOARCH}.AppImage" >/dev/null 2>&1
rm -rf "$APPDIR"
echo "✅ AppImage: ProcHub-${VERSION}-linux-${GOARCH}.AppImage"

echo ""
echo "Packaged files:"
ls -lh ProcHub-${VERSION}-linux-${GOARCH}.deb ProcHub-${VERSION}-linux-${GOARCH}.AppImage
