#!/usr/bin/env bash
# Fairwave srsRAN entrypoint — dispatches on PROFILE.
#   PROFILE=enb-zmq | ue-zmq | enb-rf   (required)
# Configs are read from RAN_CFG_DIR (default /etc/fairwave/ran).
set -euo pipefail

PROFILE="${PROFILE:?PROFILE must be set: enb-zmq | ue-zmq | enb-rf}"
RAN_CFG_DIR="${RAN_CFG_DIR:-/etc/fairwave/ran}"

# ZMQ virtual-radio plumbing (lab). The eNB and UEs share one network
# namespace (network_mode: service:enb in compose), so all ZMQ runs over
# localhost — the exact topology from the srsRAN 4G ZeroMQ app note.
# TCP transport across Docker bridge networks adds subframe jitter that
# makes the UE PHY intermittently lose sync (SYNC TRACK ret=-1) and the
# eNB to abort InitialContextSetup.
ZMQ_BASE_SRATE="${FW_ZMQ_BASE_SRATE:-23.04e6}"
ENB_TX_PORT="${FW_ENB_TX_PORT:-tcp://*:2000}"
ENB_RX_PORTS="${FW_ENB_RX_PORTS:-tcp://127.0.0.1:2001}"
UE_RX_PORT="${FW_UE_RX_PORT:-tcp://127.0.0.1:2000}"
UE_TX_PORT="${FW_UE_TX_PORT:-tcp://*:2001}"

# SIM guard: only the dummy lab test vectors (sim/test-vectors/lab-vectors.yaml)
# may ever be used by the lab UEs. No real SIMs in this stack.
lab_imsi_ok() {
    case "${1:-}" in
        999991234567001|999991234567002|999991234567003) return 0 ;;
        *) return 1 ;;
    esac
}

# Lab vector lookup: prints "KI OPC" for a lab IMSI, exits 1 otherwise.
lab_vector() {
    case "${1:-}" in
        999991234567001) echo "465B5CE8B199B49FAA5F0A2EE238A6BC 4D9B7A2C5E8F1A3B6C0D2E4F7A9B5C1D" ;;
        999991234567002) echo "8A1F3C9D5E7B2A4F6C8D0E1F3A5B7C9D 2E6B4A7D9F1C3E5B8A0D2F4C6E8B1A3F" ;;
        999991234567003) echo "3C9E5B7A1D8F2C4E6A0B3D5F7C9E1B2A 7F1D3A5C9B2E4F6A8C0D2E4B6F8A1C3" ;;
        *) return 1 ;;
    esac
}

case "$PROFILE" in

    enb-zmq)
        # Virtual cell against the Open5GS MME (see core/ran/enb.zmq.yml).
        # srsENB resolves enb_files (sib/rr/rb) relative to CWD, so cd there.
        cd "${RAN_CFG_DIR}"
        exec srsenb "enb.zmq.yml" \
            --rf.device_name=zmq \
            --rf.device_args="fail_on_disconnect=true,id=enb,base_srate=${ZMQ_BASE_SRATE},tx_port=${ENB_TX_PORT},rx_port=${ENB_RX_PORTS}" \
            --rf.time_adv_nsamples=0
        ;;

    ue-zmq)
        # A lab UE must never ride a hardware radio: refuse loudly.
        if [[ "${FAIRWAVE_RF_MODE:-}" == "hardware" ]]; then
            echo "REFUSED: PROFILE=ue-zmq is a ZMQ lab profile and must not run with FAIRWAVE_RF_MODE=hardware" >&2
            exit 1
        fi
        UE_IMSI="${UE_IMSI:-999991234567001}"
        lab_imsi_ok "$UE_IMSI" || {
            echo "REFUSED: IMSI ${UE_IMSI} is not a lab test vector (sim/test-vectors/lab-vectors.yaml). No real SIMs in the lab stack." >&2
            exit 1
        }
        read -r UE_K UE_OPC < <(lab_vector "$UE_IMSI")
        exec srsue "${RAN_CFG_DIR}/ue.zmq.yml" \
            --rf.device_name=zmq \
            --rf.device_args="fail_on_disconnect=true,id=ue,base_srate=${ZMQ_BASE_SRATE},rx_port=${UE_RX_PORT},tx_port=${UE_TX_PORT}" \
            --rf.tx_gain=80 \
            --rf.time_adv_nsamples=0 \
            --usim.imsi="${UE_IMSI}" \
            --usim.k="${UE_K}" \
            --usim.opc="${UE_OPC}"
        ;;

    enb-rf)
        # Hardware radio — gated upstream (rf-gate service / tx/arm API), but
        # defend again here so a misconfigured PROFILE can never TX silently.
        if [[ "${FAIRWAVE_RF_MODE:-}" != "hardware" ]]; then
            echo "REFUSED: PROFILE=enb-rf requires FAIRWAVE_RF_MODE=hardware. RF TX needs a country, a license acknowledgment and the band allow-list (deploy/scripts/rf-gate.sh + POST /v1/tx/arm)." >&2
            exit 1
        fi
        exec srsenb "${RAN_CFG_DIR}/enb.rf.yml"
        ;;

    *)
        echo "unknown PROFILE '${PROFILE}' (expected enb-zmq | ue-zmq | enb-rf)" >&2
        exit 1
        ;;
esac
