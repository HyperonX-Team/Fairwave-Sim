#!/usr/bin/env bash
# 5G SA attach test.
#
# Brings up the free5GC compose (core + srsRAN 5G ZMQ RAN + control plane),
# provisions the lab vector into the UDR through the hsswrite free5gc
# driver, and asserts the UE shows up in /v1/sessions (the AMF OAM
# collector sees the attach). Used by CI (.github/workflows/5g-attach.yml)
# and runnable on any host with docker.
#
# Preconditions:
#   - gtp5g kernel module loaded on the host (see core/free5gc/README.md)
#   - core/free5gc configs staged: ./core/free5gc/fetch-upstream.sh
#   - ATTACH_BUILD=0 skips the image builds (CI prebuilds with cache)
set -euo pipefail
cd "$(dirname "$0")/../.."

FW_ADMIN_TOKEN="${FW_ADMIN_TOKEN:-ci-admin}"
FW_CONTROL_PORT="${FW_CONTROL_PORT:-8080}"
ATTACH_BUILD="${ATTACH_BUILD:-1}"
COMPOSE=(docker compose -f deploy/docker-compose.5g.yml)
BASE="http://127.0.0.1:${FW_CONTROL_PORT}"
WORK="$(mktemp -d)"
IMSI=999991234567001

teardown() { "${COMPOSE[@]}" down -v >/dev/null 2>&1 || true; }
trap teardown EXIT

echo "== preflight =="
lsmod | grep -q gtp5g || { echo "FAIL: gtp5g kernel module not loaded (see core/free5gc/README.md)"; exit 1; }
[ -f core/free5gc/nrfcfg.yaml ] || { echo "FAIL: stage NF configs first: ./core/free5gc/fetch-upstream.sh"; exit 1; }

if [ "${ATTACH_BUILD}" = "1" ]; then
  echo "== building srsran image (srsgnb + srsue) =="
  ( cd deploy && docker build -f docker/Dockerfile.srsran -t fairwave/srsran:lab . )
  echo "== building control-plane image =="
  docker build -f deploy/docker/Dockerfile.control -t fairwave/control:0.1.0 .
fi

CLI="${CLI:-/tmp/bin/fairwave}"
if [ ! -x "$CLI" ]; then
  echo "== building fairwave CLI =="
  go build -o "$CLI" ./apps/fairwave-cli/cmd/fairwave
fi

echo "== starting free5GC 5G stack =="
"${COMPOSE[@]}" up -d
for i in $(seq 1 120); do
  curl -sf "${BASE}/v1/healthz" > /dev/null && break
  sleep 2
done
curl -sf "${BASE}/v1/healthz" > /dev/null || { echo "FAIL: control plane never came up"; "${COMPOSE[@]}" logs control-plane | tail -40; exit 1; }
echo "control plane up"

echo "== provisioning ${IMSI} into the UDR (hsswrite free5gc driver) =="
# --control points at a dead port so the CLI takes the standalone path,
# which runs hsswrite.New("free5gc", "mongodb") on THIS host (docker exec
# mongodb mongosh) - exercising the exact driver code against live mongo.
"$CLI" --control http://127.0.0.1:9 --data-dir "${WORK}" \
  esim issue --imsi "${IMSI}" --hss-driver free5gc --hss-container mongodb
docker exec mongodb mongosh --quiet mongodb://localhost:27017/free5gc \
  --eval "print(db[\"subscriptionData.authenticationData.authenticationSubscription\"].countDocuments({ueId: \"imsi-${IMSI}\"}))" \
  | grep -q '^1$' || { echo "FAIL: subscriber not in the UDR"; exit 1; }
echo "subscriber provisioned into the UDR"

echo "== waiting for UE attach (session in /v1/sessions) =="
WANT_HASH=$(printf '%s' "${IMSI}" | sha256sum | cut -d' ' -f1)
FOUND=0
for i in $(seq 1 120); do
  if curl -sf -H "Authorization: Bearer ${FW_ADMIN_TOKEN}" "${BASE}/v1/sessions" | grep -q "${WANT_HASH}"; then
    FOUND=1
    break
  fi
  sleep 3
done
if [ "${FOUND}" != "1" ]; then
  echo "FAIL: UE never attached"
  echo "--- control-plane sessions ---"
  curl -s -H "Authorization: Bearer ${FW_ADMIN_TOKEN}" "${BASE}/v1/sessions" || true
  echo "--- AMF log ---"
  docker logs free5gc-amf 2>&1 | grep -E "SCTP Accept|NGSetup|Registration|PDUSession" | tail -20 || true
  echo "--- gNB log ---"
  docker exec gnb cat /tmp/gnb.log 2>/dev/null | tail -30 || true
  echo "--- ue5g log ---"
  docker exec ue5g cat /tmp/ue.log 2>/dev/null | tail -30 || true
  exit 1
fi
echo "ATTACH OK: UE ${IMSI} visible in /v1/sessions"
curl -s -H "Authorization: Bearer ${FW_ADMIN_TOKEN}" "${BASE}/v1/sessions"
echo
echo "== ALL 5G ATTACH CHECKS PASSED =="
