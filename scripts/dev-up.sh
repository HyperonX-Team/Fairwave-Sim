#!/usr/bin/env bash
# dev-up.sh - bring up the full dev lab in one command.
set -euo pipefail
cd "$(dirname "$0")/.."

echo "== Fairwave dev lab (no RF) =="
make lab-up
echo
echo "== interactive helpers =="
echo "  fairwave node status        (needs control plane token)"
echo "  docker exec -it ue1 bash    (UE shell)"
