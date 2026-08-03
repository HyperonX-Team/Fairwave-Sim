---
title: Rendezvous Server
---

# Rendezvous Server: Spec and mDNS Details

Discovery in Fairwave has two mechanisms: mDNS for shared-LAN peers and an optional rendezvous server for remote or NAT'd peers. This page specifies both, including the announce format and security.

## mDNS discovery

Service type: `_fairwave._udp.local`, port 5353/udp, standard mDNS (RFC 6762).

```txt
_fairwave._udp.local PTR fairwave-a1b2c3.local
fairwave-a1b2c3.local A 192.168.1.50
fairwave-a1b2c3.local SRV 0 0 8080 fairwave-a1b2c3.local
fairwave-a1b2c3.local TXT "nodeid=a1b2c3d4" "mesh=mesh-7" "ver=0.1.0" "sig=..."
```

TXT record fields:

| Key | Meaning |
| --- | --- |
| `nodeid` | Node UUID |
| `mesh` | Mesh name (peers only answer announces with matching mesh) |
| `ver` | Fairwave version |
| `sig` | Signature over the other TXT values, node key (ed25519) |
| `ctrlport` | Control plane port (default 8080) |

**Signed announces:** every TXT record carries `sig`, an ed25519 signature over `nodeid|mesh|ver|ctrlport` with the node's discovery key. Peers cache the discovery key at join; announces that fail verification are dropped and counted. mDNS alone gives no NAT traversal - it proves reachability on the same L2 segment.

## Rendezvous server

A small standalone service (Go, `cmd/rendezvous`) that does two things:

1. **Registration and lookup** for nodes that cannot use mDNS.
2. **Hole punching assist** for WireGuard behind NAT.

### API (JSON over HTTPS, TLS 1.2+)

| Endpoint | Purpose |
| --- | --- |
| `POST /v1/announce` | Publish signed node announcement (endpoint, nodeid, mesh, pubkeys) |
| `GET /v1/lookup?mesh=...` | List current signed announcements for a mesh |
| `POST /v1/punch` | Exchange UDP endpoints between two peers (both must announce first) |
| `GET /v1/healthz` | Liveness |

Announcement payload (signed with the node's ed25519 discovery key):

```json
{
  "nodeid": "a1b2c3d4",
  "mesh": "mesh-7",
  "ver": "0.1.0",
  "endpoint": {"ip": "203.0.113.10", "udp_port": 51820},
  "seen_endpoint": "203.0.113.10:51820",
  "pubkeys": {"wg": "MjTz6B9...", "disc": "9Z4aK0..."},
  "sig": "base64..."
}
```

Rules:

- Announcements expire after 90 s; nodes must re-announce (heartbeat 30 s).
- Signed announces only; unsigned entries are rejected.
- `lookup` returns only verified, unexpired entries, rate-limited per IP.
- `punch` returns both peers' `seen_endpoint` views so each can send WireGuard handshakes to the other's mapped address. The server never relays data-plane traffic (except optional control-plane relay under explicit policy - see below).

## Security

- **Identity:** verification keys are distributed at join (mesh CA) - the rendezvous is a directory, not a trust anchor.
- **Replay:** announces carry a 60 s timestamp window; duplicates with older timestamps are dropped.
- **DoS:** per-IP announce rate limit (10/min); lookups 60/min.
- **Transparency:** announce history is logged (nodeid, mesh, endpoint hashes) - no subscriber data ever.
- **TLS:** mandatory; the rendezvous operator should pin certs in client config.

## Deployment

Reference deployment: a tiny VM or container on a stable IP, one per mesh region:

```bash
docker run -d -p 2468:2468 -p 5353:5353/udp \
  ghcr.io/hyperonx/fairwave-rendezvous:0.1.0 \
  --mesh mesh-7 --tls-cert /certs/fullchain.pem --tls-key /certs/privkey.pem
```

Operations:

- Run ≥ 2 replicas behind a load balancer; announcements are stateless.
- Back up nothing sensitive: the server holds no credentials, only transient announcements and logs.
- Monitor `GET /v1/healthz` and announce volume; alert on sudden `mesh` count drops (possible partitioning).

## When to skip it

If all nodes share a LAN, mDNS suffices. If remote peering is needed but no trusted operator wants to run a rendezvous, use direct `--rendezvous` IP configuration at join time instead of a public service - or do not peer (see [when NOT to peer](index.md#when-not-to-peer)).

## Related

- [Peering overview](index.md) · [Mesh runbook](mesh-runbook.md) · [Two-box tutorial](../tutorials/two-box-peering.md)
