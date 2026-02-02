#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BUILD_BIN="$ROOT_DIR/dist/dropserve"
INSTALL_DIR="$HOME/.local/bin"
INSTALL_PATH="$INSTALL_DIR/dropserve"

if [ ! -f "$BUILD_BIN" ]; then
  printf 'Error: built binary not found at %s\n' "$BUILD_BIN" >&2
  printf 'Run: %s\n' "$ROOT_DIR/scripts/build.sh" >&2
  exit 1
fi

mkdir -p "$INSTALL_DIR"
install -m 0755 "$BUILD_BIN" "$INSTALL_PATH"

printf 'Installed binary: %s\n' "$INSTALL_PATH"
