#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if ! command -v sql-migrate >/dev/null 2>&1; then
  echo "error: sql-migrate is not installed" >&2
  exit 1
fi

sql-migrate down -limit=1 -config "$ROOT_DIR/backend/dbconfig.yml" -env development
