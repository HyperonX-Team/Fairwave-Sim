---
title: Mesh Runbook
---

# Mesh Runbook: Setup, Keys, Routes, NAT, Failures

Operational reference for building and maintaining a Fairwave mesh. Follow the [two-box tutorial](../tutorials/two-box-peering.md) once before treating this as the reference.

## Bootstrap tokens

Joining requires a bootstrap token minted on an already-trusted node:

```bash
fairwave node join --generate-token --ttl 15m --role peer
```

Properties:

- One-time use: the token is consumed by the first successful join; re-issuing is explicit.
- TTL: default 15 minutes (`--ttl`), max 24 h enforced by policy.
- Scoped: `--role` limits what the joining node may do (default `peer`; `peer+route-advertise` only with `sim:admin` sign-off).

On the joining node:

```bash
fairwave node join --token fw-join-...
```

Expected:

```
peer site-b added to mesh
  control plane: mTLS established (mesh CA chain)
  wireguard:     handshake completed
  routes:        advertise 10.45.0.0/24; accept site-a 10.44.0.0/24
```

## WireGuard keys

- Keypair generated at `node init`; stored in the node's key store (0600), never in config files.
- Public keys are exchanged over the mTLS control channel at join - never manually pasted.
- Rotation: `fairwave node wg-rotate` re-keys all peers in one pass; the old key is kept for one keepalive window (25 s) to avoid blackholes.

```bash
fairwave node wg-rotate
```

```
wg key rotated: 12 peers updated, 0 unreachable
```

## Allowed IPs and route exchange

Each node advertises:

- Its UE pool (PGW subnet, e.g. `10.44.0.0/24`).
- Optional site prefixes (`--advertise 192.168.7.0/24`), policy-gated.

The agent merges all peers' advertisements into WireGuard `AllowedIPs`. Conflicts (overlapping subnets from two peers) are rejected at the control plane and surfaced in `fairwave peer list`:

```
WARN route conflict: 10.44.0.0/24 from site-c overlaps site-a; ignored
```

Per-peer overrides are possible but discouraged: `fairwave node peer-set --peer site-c --allow 10.44.0.0/25`.

## NAT traversal

WireGuard speaks UDP (`51820/udp`), so:

- **Same LAN:** mDNS discovery, direct connection, no traversal needed.
- **Both behind NAT:** peer-to-peer won't connect by itself. Options:
  1. Rendezvous server performs UDP hole punching (both sides punch to the server's seen endpoints); works for most consumer NATs.
  2. If punching fails, the rendezvous can relay control-plane traffic only; data stays direct - do not fall back to relaying subscriber data without policy review.

```bash
fairwave node join --token ... --rendezvous rdz.example.net:2468
```

- **Preserve the NAT mapping**: keepalives every 25 s; `PersistentKeepalive` on both sides.

## Failure handling

| Symptom | Diagnosis | Action |
| --- | --- | --- |
| `peer list` shows unreachable | Keepalives lost > 180 s | Wait for renegotiation; check NAT mapping, firewall 51820/udp |
| wg handshake timeout | Keys or endpoint stale | `fairwave doctor`; `fairwave peer list`; re-run join if certs revoked |
| Routes withdrawn | Peer unreachable or config conflict | Traffic falls back to local breakout - verify PGW NAT still up |
| Peer re-joins with new identity | Reboot/re-init | Control plane rejects same-node-id with different pubkey; re-issue token |
| Split brain (two CAs) | Manual misconfig | Re-join node to the authoritative mesh; stale CA certs rotate |
| Rendezvous down | Discovery degrades | mDNS still works on LAN; direct joins use `--rendezvous` only at join |

## Operational hygiene

- Revoke peers promptly: `fairwave node revoke --peer site-b --reason decommissioned`.
- Rotate mesh CA: annually or on any credential compromise; `fairwave mesh ca-rotate` re-signs all nodes (rolling rejoin).
- Monitor: `/v1/peers` exposes per-peer state, handshake age, and last route update - dashboards should alarm on `unreachable > 5 min` for non-leaf nodes.
- Test failover quarterly: kill a peer's WireGuard endpoint and confirm traffic reroutes or falls back within 180 s.

## Related

- [Peering overview](index.md) · [Rendezvous](rendezvous.md) · [Two-box tutorial](../tutorials/two-box-peering.md) · ADR-0004
