#!/usr/bin/env bash
# Fairwave Hydra local demo - no Docker, no radio.
#
# Starts a control plane, registers a three-thread weave (three SIMs on
# three boxes), runs the speedup bench, and serves the operator dashboard.
# Everything is loopback UDP: nothing transmits, nothing touches your real
# network or Wi-Fi.
#
#   bash scripts/hydra-demo.sh          # then open the printed URL
#   Ctrl-C to stop
#
# Environment overrides: FAIRWAVE_DEMO_PORT (default 8090),
# FAIRWAVE_DEMO_TOKEN (default demo-token), FAIRWAVE_DEMO_DIR (default .demo).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

PORT="${FAIRWAVE_DEMO_PORT:-8090}"
TOKEN="${FAIRWAVE_DEMO_TOKEN:-demo-token}"
BASE="http://127.0.0.1:${PORT}"
DEMO_DIR="${FAIRWAVE_DEMO_DIR:-$ROOT/.demo}"
LOG="$DEMO_DIR/control.log"

mkdir -p "$DEMO_DIR" bin
go build -o bin/fairwave-control ./core/fairwave-control/cmd/fairwave-control
go build -o bin/fairwave ./apps/fairwave-cli/cmd/fairwave

cleanup() {
  if [[ -n "${SRV:-}" ]]; then kill "$SRV" 2>/dev/null || true; fi
}
trap cleanup EXIT

echo "starting control plane on ${BASE} ..."
FAIRWAVE_ADMIN_TOKEN="$TOKEN" \
FAIRWAVE_SERVER_LISTEN="127.0.0.1:${PORT}" \
FAIRWAVE_SERVER_DATADIR="${DEMO_DIR}/data" \
FAIRWAVE_SERVER_UIDIR="${ROOT}/apps/fairwave-ui" \
FAIRWAVE_SERVER_MODE=lab \
FAIRWAVE_HYDRA_ANCHOR=demo-hub \
  ./bin/fairwave-control >"$LOG" 2>&1 &
SRV=$!

ready=0
for _ in $(seq 1 50); do
  if curl -sf "${BASE}/v1/healthz" >/dev/null 2>&1; then ready=1; break; fi
  sleep 0.2
done
if [[ "$ready" != 1 ]]; then
  echo "control plane failed to start:"
  cat "$LOG"
  exit 1
fi

FW=(./bin/fairwave --control "$BASE" --token "$TOKEN")

echo "registering a three-thread weave (3 SIMs on 3 boxes, 300 Mbps each) ..."
"${FW[@]}" hydra thread-add --id sim-a --box box-a --mbps 300 --rtt 18 --up >/dev/null
"${FW[@]}" hydra thread-add --id sim-b --box box-b --mbps 300 --rtt 22 --up >/dev/null
"${FW[@]}" hydra thread-add --id sim-c --box box-c --mbps 300 --rtt 30 --up >/dev/null
"${FW[@]}" hydra weave-create --id demo --anchor demo-hub --threads sim-a,sim-b,sim-c >/dev/null
"${FW[@]}" hydra bench demo | sed 's/^/  /'

cat <<EOF

  ------------------------------------------------------------------
   Fairwave dashboard:  ${BASE}/
   Token to paste:      ${TOKEN}   (click "Set API token")

   One box alone = 300 Mbps.  This weave = 900 Mbps (3.00x).

   Try the CLI too:
     ./bin/fairwave --control ${BASE} --token ${TOKEN} hydra status
     ./bin/fairwave --control ${BASE} --token ${TOKEN} hydra bench demo

   Loopback only: no radio, no RF, nothing on your real network.
   Press Ctrl-C to stop.
  ------------------------------------------------------------------
EOF

wait "$SRV"
