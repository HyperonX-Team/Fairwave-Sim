#!/usr/bin/env bash
# Hydra end-to-end smoke test.
#
# Exercises the real `fairwave-control` daemon and the real `fairwave` CLI
# over HTTP: seed threads, weave them, run the bench, strip/ingest a frame,
# and assert the aggregate ceiling is the sum of the threads. No Docker, no
# RF, no ZMQ - safe for CI on any runner with Go and curl.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

PORT="${FW_HYDRA_SMOKE_PORT:-18080}"
TOKEN="${FW_HYDRA_SMOKE_TOKEN:-hydra-smoke-token}"
BASE="http://127.0.0.1:${PORT}"
TMP="$(mktemp -d)"
LOG="$TMP/control.log"

cleanup() {
  if [[ -n "${SRV:-}" ]]; then kill "$SRV" 2>/dev/null || true; fi
  rm -rf "$TMP"
}
trap cleanup EXIT

echo "== build =="
mkdir -p bin
go build -o bin/fairwave-control ./core/fairwave-control/cmd/fairwave-control
go build -o bin/fairwave ./apps/fairwave-cli/cmd/fairwave

echo "== start control plane on ${BASE} =="
FAIRWAVE_ADMIN_TOKEN="$TOKEN" \
FAIRWAVE_SERVER_LISTEN="127.0.0.1:${PORT}" \
FAIRWAVE_SERVER_DATADIR="${TMP}/data" \
FAIRWAVE_SERVER_MODE=lab \
FAIRWAVE_HYDRA_ANCHOR=smoke-anchor \
  ./bin/fairwave-control >"$LOG" 2>&1 &
SRV=$!

ready=0
for _ in $(seq 1 50); do
  if curl -sf "${BASE}/v1/healthz" >/dev/null 2>&1; then ready=1; break; fi
  sleep 0.2
done
if [[ "$ready" != 1 ]]; then
  echo "FAIL: control plane did not become healthy"
  cat "$LOG"
  exit 1
fi

FW=(./bin/fairwave --control "$BASE" --token "$TOKEN")

echo "== seed three threads (three SIMs on three boxes) =="
"${FW[@]}" hydra thread-add --id smoke-a --box box-a --mbps 300 --rtt 18 --up
"${FW[@]}" hydra thread-add --id smoke-b --box box-b --mbps 300 --rtt 22 --up
"${FW[@]}" hydra thread-add --id smoke-c --box box-c --mbps 300 --rtt 30 --up

echo "== weave them into one logical link =="
"${FW[@]}" hydra weave-create --id smoke --anchor smoke-anchor --threads smoke-a,smoke-b,smoke-c

echo "== bench: the ceiling becomes the sum =="
BENCH="$("${FW[@]}" hydra bench smoke)"
echo "$BENCH"
echo "$BENCH" | grep -q "speedup:        3.00x" || { echo "FAIL: expected a 3.00x speedup"; exit 1; }
echo "$BENCH" | grep -q "aggregate:      900 Mbps" || { echo "FAIL: expected a 900 Mbps aggregate"; exit 1; }
echo "$BENCH" | grep -q "delivered:      300" || { echo "FAIL: expected 300 delivered packets"; exit 1; }

echo "== status =="
"${FW[@]}" hydra status | tee "${TMP}/status.txt"
grep -q "aggregate:        900 Mbps" "${TMP}/status.txt" || { echo "FAIL: status aggregate"; exit 1; }
grep -q "payloads delivered: 300" "${TMP}/status.txt" || { echo "FAIL: status delivered"; exit 1; }

echo "== strip / ingest round trip =="
FRAME="$("${FW[@]}" hydra strip smoke --payload hello-hydra | tail -n 1)"
"${FW[@]}" hydra ingest smoke --frame "$FRAME" | grep -q "delivered 1" || { echo "FAIL: ingest"; exit 1; }

echo "== json status reflects hydra =="
curl -sf -H "Authorization: Bearer ${TOKEN}" "${BASE}/v1/status" \
  | grep -q '"hydra_aggregate_mbps":900' || { echo "FAIL: /v1/status hydra fields"; exit 1; }

echo "== cleanup =="
"${FW[@]}" hydra weave-remove smoke
"${FW[@]}" hydra thread-remove smoke-a

echo "HYDRA SMOKE OK"
