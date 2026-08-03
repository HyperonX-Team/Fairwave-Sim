#!/usr/bin/env bash
# release.sh — cut a Fairwave release: tag, changelog, SBOM, sign.
# Usage: ./scripts/release.sh 0.2.0
set -euo pipefail
VERSION="${1:-0.1.0}"

cd "$(dirname "$0")/.."

echo "== Fairwave release $VERSION =="

# 1. changelog sanity
[ -f CHANGELOG.md ] || { echo "CHANGELOG.md missing"; exit 1; }
grep -q "## \[$VERSION\]" CHANGELOG.md || { echo "no CHANGELOG entry for $VERSION"; exit 1; }

# 2. tags
if git rev-parse "v$VERSION" >/dev/null 2>&1; then
  echo "tag v$VERSION already exists"
else
  git tag -s "v$VERSION" -m "Fairwave v$VERSION (lab release)"
  echo "tagged v$VERSION"
fi

# 3. SBOM
if command -v syft >/dev/null 2>&1; then
  syft dir:. --output spdx-json --file "sbom-v$VERSION.spdx.json"
  echo "SBOM written"
fi

# 4. sign (keyless cosign when available; operator may skip in offline envs)
if command -v cosign >/dev/null 2>&1; then
  cosign attest-blob --yes "sbom-v$VERSION.spdx.json" || echo "[warn] cosign attest failed (keys not configured?)"
fi

echo "== done. Push tag: git push origin v$VERSION =="
