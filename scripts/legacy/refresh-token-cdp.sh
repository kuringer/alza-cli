#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if ! command -v bun >/dev/null 2>&1; then
  echo "bun not found. Install bun first."
  exit 1
fi

exec bun "$SCRIPT_DIR/refresh-token-cdp.mjs"
