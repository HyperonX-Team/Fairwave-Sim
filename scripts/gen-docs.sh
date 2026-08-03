#!/usr/bin/env bash
# gen-docs.sh - generate derived docs (API reference) into docs/_gen/.
set -euo pipefail
cd "$(dirname "$0")/.."

GEN=docs/_gen
mkdir -p "$GEN"

# OpenAPI: keep a curated copy of the API surface up to date.
if command -v npx >/dev/null 2>&1 && [ -f api/openapi.yaml ]; then
  npx --yes @redocly/cli@latest build-docs api/openapi.yaml -o "$GEN/openapi.html" \
    || echo "[warn] redocly failed; docs/_gen/openapi.html is a copy placeholder"
fi

echo "generated docs in $GEN"
