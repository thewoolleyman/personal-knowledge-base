#!/usr/bin/env bash
# validate-release.sh - Validate release artifacts by installing and running the current-platform binary.
# Usage: scripts/validate-release.sh [VERSION]
# Expects dist/ to contain zip files from build-release.sh.
# Installs the zip for the current OS/arch into a temp dir and runs "pkb version".
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="${1:-$(tr -d '[:space:]' < "$REPO_ROOT/VERSION")}"
DIST_DIR="$REPO_ROOT/dist"

GOOS="$(go env GOOS)"
GOARCH="$(go env GOARCH)"

ARCHIVENAME="pkb-${GOOS}-${GOARCH}-v${VERSION}"
ZIPFILE="$DIST_DIR/${ARCHIVENAME}.zip"

if [ ! -f "$ZIPFILE" ]; then
  echo "FAIL: Expected release artifact not found: $ZIPFILE" >&2
  exit 1
fi

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

echo "Validating $ZIPFILE..."
unzip -o "$ZIPFILE" -d "$TMPDIR"

BINARY="$TMPDIR/$ARCHIVENAME/pkb"
if [ "$GOOS" = "windows" ]; then
  BINARY="$TMPDIR/$ARCHIVENAME/pkb.exe"
fi

if [ ! -f "$BINARY" ]; then
  echo "FAIL: Binary not found in zip: $BINARY" >&2
  exit 1
fi

chmod +x "$BINARY"

# Verify the binary runs and reports the expected version
OUTPUT="$("$BINARY" version 2>&1)"
echo "Binary output: $OUTPUT"

if echo "$OUTPUT" | grep -q "$VERSION"; then
  echo "OK: Binary reports version $VERSION"
else
  echo "FAIL: Expected version $VERSION in output, got: $OUTPUT" >&2
  exit 1
fi

echo "Validation passed."
