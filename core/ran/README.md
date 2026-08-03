# Radio Access Network configs (4G)

Lab-first: the default path is **ZMQ virtual radio** — srsENB and srsUE run
in containers and exchange IQ over `tcp://` on the compose network. There is
no antenna, no gain, no licensed spectrum involvement. This is the config
every new contributor starts with (`docker compose -f deploy/docker-compose.yml up`).

| File | Device | Purpose | Gates |
| --- | --- | --- | --- |
| `enb.zmq.yml` | srsENB + ZMQ | Virtual cell (PLMN 999-99, TAC 7, EARFCN 3350) | none (lab) |
| `ue.zmq.yml` | srsUE + ZMQ | Virtual handset using dummy vector 999991234567001 | IMSI allow-list in `srs-entry.sh` |
| `enb.rf.yml` | srsENB + UHD (USRP) | Hardware template — frequency/gain per license | `FAIRWAVE_RF_MODE=hardware` + ack file + band allow-list |

## How the pieces connect

- srsENB S1-MME → Open5GS MME (`mme_addr`, default `10.10.0.2` on the fwnet
  subnet). Verify the container IP if you change the compose subnet.
- srsENB S1-U → Open5GS SGW (internal to the EPC container).
- srsUE → srsENB over ZMQ ports 2000/2001. `deploy/docker/srs-entry.sh`
  injects the device args; two UEs share uplink via a comma-separated
  `rx_port` list on the eNB (one PULL sink, multiple PUSH sources).
- HSS holds the Ki/OPc; the UE config embeds the dummy vector for a fast
  attach path. Lab IMSIs are allow-listed in `srs-entry.sh` — anything else
  is refused, on purpose.

## RF (hardware) path — honest note on RU/DU

Hardware TX requires the full gate: `FAIRWAVE_RF_MODE=hardware`, a country
file, a license acknowledgment file, and a band allow-list check through
`POST /v1/spectrum/check` before `POST /v1/tx/arm` may succeed. Today the
radio is a **monolithic srsENB**: PHY, MAC, RLC, PDCP and S1AP run in one
process, which is the pragmatic choice for a community small cell. A future
**O-RAN 7.2 split** (RU/DU separation over the eCPRI/7.2 interface) is on the
roadmap — it would let Fairwave drive third-party RUs instead of shipping a
radio — but it is a substantial project (7.2 payload encoding, timing
synchronization between RU and DU, and a FAPI-to-7.2 bridge), and the
monolithic PHY remains the supported configuration until then. Do not plan a
deployment on the split.
