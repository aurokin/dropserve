#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

bun run --cwd "$ROOT_DIR/web" build
go run "$ROOT_DIR/cmd/dropserve" serve
