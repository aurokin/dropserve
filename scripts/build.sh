#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if ! command -v bun >/dev/null 2>&1; then
  printf 'Error: bun is not installed or not on PATH.\n' >&2
  exit 1
fi

if ! command -v go >/dev/null 2>&1; then
  printf 'Error: go is not installed or not on PATH.\n' >&2
  exit 1
fi

printf 'Building web assets...\n'
cd "$ROOT_DIR/web"
bun install
bun run build

printf 'Building Go binary...\n'
cd "$ROOT_DIR"
mkdir -p dist
go build -o dist/dropserve ./cmd/dropserve

printf 'Built binary: %s\n' "$ROOT_DIR/dist/dropserve"
