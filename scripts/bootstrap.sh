#!/usr/bin/env bash
# bootstrap.sh - install/verify Fairwave toolchains (idempotent).
set -euo pipefail

echo "== Fairwave bootstrap =="

has() { command -v "$1" >/dev/null 2>&1; }

# --- Go ---
if has go; then
  echo "[ok] go $(go version | awk '{print $3}')"
else
  echo "[install] go 1.22+ (download from https://go.dev/dl or your package manager)"
fi

# --- Docker ---
if has docker; then
  echo "[ok] docker $(docker --version 2>/dev/null | awk '{print $3}')"
  docker info >/dev/null 2>&1 && echo "[ok] docker daemon running" \
    || echo "[warn] docker daemon not running; start it before make lab-up"
else
  echo "[install] Docker Engine 24+ (https://docs.docker.com/engine/install/)"
fi

# --- GNU make / just ---
has make && echo "[ok] make" || echo "[install] GNU make"
has just && echo "[ok] just" || echo "[skip] just (optional)"

# --- pre-commit ---
if has pre-commit; then
  echo "[ok] pre-commit $(pre-commit --version)"
  pre-commit install 2>/dev/null || true
else
  echo "[install] pre-commit: pip install pre-commit"
fi

# --- syft (SBOM) ---
has syft && echo "[ok] syft" || echo "[install] syft: https://github.com/anchore/syft (optional until release)"

# --- cosign (release signing) ---
has cosign && echo "[ok] cosign" || echo "[install] cosign: https://github.com/sigstore/cosign (optional until release)"

# --- golangci-lint ---
if has golangci-lint; then
  echo "[ok] golangci-lint"
else
  echo "[install] golangci-lint: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
fi

# --- Python (docs) ---
has python3 && echo "[ok] python3 $(python3 --version 2>&1 | awk '{print $2}')" || echo "[install] python3 for mkdocs"

echo
echo "Bootstrap complete. Next:"
echo "  make lab-up    # full no-RF lab (needs docker daemon)"
echo "  make docs-serve  # local docs site"
