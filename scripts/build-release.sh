#!/usr/bin/env bash
# build-release.sh - Cross-compile pkb for all supported platforms and package as zip files.
# Usage: scripts/build-release.sh [VERSION]
# VERSION defaults to contents of VERSION file.
# Output: dist/pkb-<os>-<arch>-v<version>.zip for each target platform.
# Each zip extracts to a subdirectory named pkb-<os>-<arch>-v<version>/ containing the binary.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="${1:-$(tr -d '[:space:]' < "$REPO_ROOT/VERSION")}"
DIST_DIR="$REPO_ROOT/dist"

TARGETS="${TARGETS:-linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64}"

rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

for target in $TARGETS; do
  GOOS="${target%/*}"
  GOARCH="${target#*/}"
  ext=""
  if [ "$GOOS" = "windows" ]; then ext=".exe"; fi
  binname="pkb${ext}"

  echo "Building ${GOOS}/${GOARCH}..."
  CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build \
    -ldflags "-X main.version=$VERSION" \
    -o "$DIST_DIR/$binname" "$REPO_ROOT/cmd/pkb"

  # Ad-hoc sign macOS binaries when codesign is available (macOS runners)
  if [ "$GOOS" = "darwin" ] && command -v codesign &>/dev/null; then
    echo "Ad-hoc signing ${GOOS}/${GOARCH} binary..."
    codesign -s - --force "$DIST_DIR/$binname"
  fi

  archivename="pkb-${GOOS}-${GOARCH}-v${VERSION}"
  zipname="$DIST_DIR/${archivename}.zip"
  mkdir -p "$DIST_DIR/$archivename"
  mv "$DIST_DIR/$binname" "$DIST_DIR/$archivename/$binname"
  (cd "$DIST_DIR" && zip -r "$zipname" "$archivename" && rm -rf "$archivename")
done

echo "Release artifacts in $DIST_DIR:"
ls -la "$DIST_DIR"
