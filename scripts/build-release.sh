#!/usr/bin/env bash
# build-release.sh - Cross-compile pkb for all supported platforms and package as zip files.
# Usage: scripts/build-release.sh [VERSION]
# VERSION defaults to contents of VERSION file.
# Output: dist/pkb-<os>-<arch>.zip for each target platform.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="${1:-$(tr -d '[:space:]' < "$REPO_ROOT/VERSION")}"
DIST_DIR="$REPO_ROOT/dist"

TARGETS="linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64"

rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

for target in $TARGETS; do
  GOOS="${target%/*}"
  GOARCH="${target#*/}"
  ext=""
  if [ "$GOOS" = "windows" ]; then ext=".exe"; fi
  binname="pkb${ext}"

  echo "Building ${GOOS}/${GOARCH}..."
  GOOS=$GOOS GOARCH=$GOARCH go build \
    -ldflags "-X main.version=$VERSION" \
    -o "$DIST_DIR/$binname" "$REPO_ROOT/cmd/pkb"

  zipname="$DIST_DIR/pkb-${GOOS}-${GOARCH}.zip"
  (cd "$DIST_DIR" && zip -j "$zipname" "$binname" && rm "$binname")
done

echo "Release artifacts in $DIST_DIR:"
ls -la "$DIST_DIR"
