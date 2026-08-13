# Radio Access Network configs (4G + 5G SA)

Lab-first: the default path is **ZMQ virtual radio** - the RAN binaries run
in containers and exchange IQ over `tcp://` on localhost (the UE shares the
cell's network namespace). There is no antenna, no gain, no licensed
spectrum involvement.

| File | Device | Core | Purpose | Gates |
| --- | --- | --- | --- | --- |
| `enb.zmq.yml` | srsENB + ZMQ | Open5GS (4G) | Virtual cell (PLMN 999-99, TAC 7, EARFCN 3350) | none (lab) |
| `ue.zmq.yml` | srsUE + ZMQ | Open5GS (4G) | Virtual handset using dummy vector 999991234567001 | IMSI allow-list in `srs-entry.sh` |
| `gnb.zmq.yml` | srsGNB + ZMQ | free5GC (5G SA) | 5G cell (PLMN 999-99, TAC 1, n3, 20 MHz, slice 010203) - `deploy/docker-compose.5g.yml` | none (lab) |
| `ue5g.zmq.yml` | srsUE 5G SA + ZMQ | free5GC (5G SA) | 5G handset (dummy vector 999991234567001) | IMSI allow-list in `srs-entry.sh` |
| `enb.rf.yml` | srsENB + UHD (USRP) | Open5GS (4G) | Hardware template - frequency/gain per license | `FAIRWAVE_RF_MODE=hardware` + ack file + band allow-list |

## How the pieces connect

- **4G**: srsENB S1-MME → Open5GS MME (`mme_addr`, default `10.10.0.2` on
  the fwnet subnet); S1-U → Open5GS SGW (internal to the EPC container).
- **5G SA**: srsGNB N2 → free5GC AMF (10.100.200.16:38412), N3 → free5GC
  UPF; the cell broadcasts PLMN 999-99, TAC 1 and slice SST 1 / SD 010203
  (all agreed with `core/free5gc/*.yaml` and the SIM vectors).
- srsUE → srsENB/srsGNB over ZMQ ports 2000/2001. `deploy/docker/srs-entry.sh`
  injects the device args / CLI usim overrides; the UE container shares the
  cell's network namespace so ZMQ runs over localhost (one PULL sink, one
  PUSH source - the same topology as the srsRAN ZMQ app notes).
- HSS (Open5GS) / UDR (free5GC) holds the Ki/OPc; the UE config embeds the
  dummy vector for a fast attach path. Lab IMSIs are allow-listed in
  `srs-entry.sh` - anything else is refused, on purpose.

## 5G SA notes

- The 5G UE ships inside **srsRAN_4G** (srsRAN Project is a CU/DU and has
  no UE app); the pinned srsRAN_4G commit (`ec29b0c1f`) is the version
  proven against free5GC in the reference ZMQ lab. The gNB is
  **srsRAN_Project** (built with `-DENABLE_ZEROMQ=ON`).
- The slice SD is **decimal** in both RAN configs: 0x010203 = 66051.
- The ZMQ gNB needs CPU headroom (the reference lab reports 5 cores for
  reliable UE attach) - on 4-core CI runners the attach can be slow.

## RF (hardware) path - honest note on RU/DU

Hardware TX requires the full gate: `FAIRWAVE_RF_MODE=hardware`, a country
file, a license acknowledgment file, and a band allow-list check through
`POST /v1/spectrum/check` before `POST /v1/tx/arm` may succeed. Today the
radio is a **monolithic srsENB**: PHY, MAC, RLC, PDCP and S1AP run in one
process, which is the pragmatic choice for a community small cell. A future
**O-RAN 7.2 split** (RU/DU separation over the eCPRI/7.2 interface) is on the
roadmap - it would let Fairwave drive third-party RUs instead of shipping a
radio - but it is a substantial project (7.2 payload encoding, timing
synchronization between RU and DU, and a FAPI-to-7.2 bridge), and the
monolithic PHY remains the supported configuration until then. Do not plan a
deployment on the split.
