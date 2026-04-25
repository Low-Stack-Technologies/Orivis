#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "error: required tool '$1' is not installed" >&2
    exit 1
  fi
}

require_cmd oapi-codegen
require_cmd sqlc
require_cmd npm

echo "==> Generating Go API models and server interfaces"
mkdir -p "$ROOT_DIR/backend/internal/apigen"
oapi-codegen -config "$ROOT_DIR/backend/api/oapi-codegen.yaml" "$ROOT_DIR/openapi/orivis.openapi.yaml"

echo "==> Generating SQLc models and query code"
mkdir -p "$ROOT_DIR/backend/internal/db"
sqlc generate -f "$ROOT_DIR/backend/sqlc.yaml"

echo "==> Generating frontend Orval types and React Query hooks"
npm --prefix "$ROOT_DIR/frontend" run generate:api

echo "Done. Code generation complete."
