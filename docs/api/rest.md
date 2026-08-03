---
title: REST API Reference
---

# REST API Reference (v1)

Canonical contract: `api/openapi.yaml`. All paths below assume base `http://127.0.0.1:8080` (lab). Auth: mTLS or bearer JWT (see [API overview](index.md)). Errors: `{"error":{"code":"...","message":"..."}}`.

## Endpoint table

| Method | Path | Purpose | Auth (min role) |
| --- | --- | --- | --- |
| GET | `/v1/healthz` | Liveness/readiness | none |
| GET | `/v1/version` | Build version info | none |
| GET | `/v1/status` | Node + EPC state summary | viewer |
| GET | `/v1/nodes` | Peer/node registry | viewer |
| GET | `/v1/sims` | List SIM metadata (hashes) | viewer |
| POST | `/v1/sims` | Issue SIMs (provisioner) | sim:issuer |
| GET | `/v1/peers` | Mesh peers, routes, health | viewer |
| GET | `/v1/sessions` | Active sessions (hashes) | viewer |
| GET | `/v1/policy` | Current policy bundle | viewer |
| PUT | `/v1/policy` | Update policy | operator |
| POST | `/v1/spectrum/check` | Validate frequency/band profile | operator |
| GET | `/v1/tx/arm` | TX gate state (3 layers) | viewer |
| POST | `/v1/tx/arm` | Arm/disarm TX | sim:admin |
| POST | `/v1/lifecycle/transition` | provision→register→on-air→peer→breakout | operator |
| GET | `/metrics` | Prometheus metrics | none (network-scoped) |

## Examples

### `GET /v1/healthz`

```json
{"status":"ok","version":"0.1.0","components":{"open5gs":"ok","enb":"ok","ue":"ok"}}
```

### `GET /v1/version`

```json
{"version":"0.1.0","milestone":"M2","build":"20260802.01","commit":"a1b2c3d","go":"go1.22.4"}
```

### `GET /v1/status`

```json
{
  "state": "on-air",
  "plmn": "999-99",
  "tac": 7,
  "apns": ["internet", "ims"],
  "enb": {"connected": true, "mode": "zmq"},
  "ue": {"registered": true, "sim_state": "activated"},
  "breakout": {"mode": "local", "peer_count": 0}
}
```

`state` values: `bootstrap`, `registered`, `on-air`, `peered`, `degraded`.

### `GET /v1/nodes`

```json
{
  "nodes": [
    {"node_id": "a1b2c3d4-...", "name": "site-a", "state": "on-air",
     "plmn": "999-99", "last_seen": "2026-08-02T11:00:00Z"}
  ]
}
```

### `GET /v1/sims`

```json
{
  "sims": [
    {"imsi_sha256_12": "9f2c41b07d3a", "profile": "lab", "state": "in-use",
     "apns": ["internet", "ims"], "expires_at": "2026-08-09T09:00:00Z"}
  ],
  "next_cursor": null
}
```

### `POST /v1/sims` — issue

Request:

```json
{"profile": "lab", "count": 1, "prefix": "9999912", "label": "first-sim"}
```

Response `201`:

```json
{"issued": 1, "output_dir": "sims/2026-08-02",
 "first": {"imsi_sha256_12": "9f2c41b07d3a", "expires_at": "2026-08-09T09:00:00Z"}}
```

### `GET /v1/peers`

```json
{
  "peers": [
    {"name": "site-b", "state": "active", "endpoint": "wg://site-b:51820",
     "handshake_age_s": 12, "routes": ["10.45.0.0/24"]}
  ]
}
```

### `GET /v1/sessions`

```json
{
  "sessions": [
    {"imsi_sha256_12": "9f2c41b07d3a", "apn": "internet", "ue_ip": "10.45.0.2",
     "state": "active", "started_at": "2026-08-02T10:12:00Z"}
  ]
}
```

### `GET /v1/policy`

```json
{
  "retention_days": {"sessions": 30, "audit": 400, "metrics": 90},
  "rate_limits": {"sim_issue_per_hour": 500, "sim_revoke_per_hour": 200},
  "sim_swap_cooldown_hours": 24,
  "mesh": {"allow_route_advertise": false}
}
```

`PUT /v1/policy` accepts the same shape; changed keys only, validated against schema (ADR-0012).

### `POST /v1/spectrum/check`

Request:

```json
{"profile_id": "us-gaa-b48", "country": "US", "earfcn": 55090}
```

Response:

```json
{"allowed": false, "gate": "allow-list",
 "reason": "earfcn 55090 not in armed allow-list for country US",
 "channels_checked": 1}
```

`allowed` requires all three layers: country code set, license acknowledged, frequency in allow-list (ADR-0008).

### `GET /v1/tx/arm`

```json
{
  "armed": false,
  "gates": {"country_code": "set", "license_ack": "set", "allow_list": "missing"},
  "profile": null
}
```

`POST /v1/tx/arm` request: `{"arm": true, "profile_id": "us-gaa-b48"}` — requires all gates green, else `409`:

```json
{"error": {"code": "tx_gate_blocked", "message": "allow-list missing earfcn 55090"}}
```

### `POST /v1/lifecycle/transition`

Request: `{"from": "register", "to": "on-air", "force": false}`

Response:

```json
{"transitioned": "on-air", "at": "2026-08-02T11:05:00Z"}
```

Invalid transition → `409` with `code: invalid_transition`.

### `GET /metrics`

Prometheus text format; labels by state/apn/plmn only (no subscriber identifiers):

```text
fairwave_sessions_active{apn="internet",plmn="999-99"} 3
fairwave_sim_state{state="in-use"} 4
fairwave_attach_total{plmn="999-99"} 12
```

## Error example (common shape)

```json
{
  "error": {
    "code": "rate_limited",
    "message": "sim issue quota exceeded: 501/500 per hour"
  }
}
```

## Rate limits

Per principal per hour: `sim_issue` 500, `sim_revoke` 200, `spectrum_check` 300, `tx_arm` 60. Excess → `429`. All limits configurable via `/v1/policy`.
