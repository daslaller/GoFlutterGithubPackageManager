#!/usr/bin/env bash
set -euo pipefail

# Bootstrap installer for flutter-pm (pre-release)
# This script downloads the latest release artifact for your OS/ARCH
# and places 'flutter-pm' into ~/.local/bin (or a directory on PATH), then runs it.

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) echo "Unsupported arch: $ARCH" >&2; exit 1 ;;
 esac

BIN_DIR="$HOME/.local/bin"
mkdir -p "$BIN_DIR"

# Download from GitHub Releases (latest)
URL="https://github.com/daslaller/GoFlutterGithubPackageManager/releases/latest/download/flutter-pm_${OS}_${ARCH}.tar.gz"
echo "Downloading flutter-pm from $URL ..."
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

# Requires curl and tar
if ! command -v curl >/dev/null 2>&1; then echo "curl is required" >&2; exit 1; fi
if ! command -v tar >/dev/null 2>&1; then echo "tar is required" >&2; exit 1; fi

curl -fsSL "$URL" -o "$TMP/fpm.tgz"
tar -xzf "$TMP/fpm.tgz" -C "$TMP"
install -m 0755 "$TMP/flutter-pm" "$BIN_DIR/flutter-pm"

if ! command -v flutter-pm >/dev/null 2>&1; then
  echo "Add $BIN_DIR to your PATH (e.g., export PATH=\"$BIN_DIR:$PATH\")."
else
  echo "Installed flutter-pm to $BIN_DIR."
fi

exec "$BIN_DIR/flutter-pm" "$@"
