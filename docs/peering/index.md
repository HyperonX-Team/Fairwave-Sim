---
title: Peering Overview
---

# Peering: Meshing Fairwave Nodes

Peering connects Fairwave nodes into a private mesh so subscriber traffic can cross node boundaries - a UE on node A reaching a service behind node B without going through the public internet. It is optional, opt-in, and off by default.

> Fairwave defaults to lab/no-RF mode. Transmitting on cellular bands without proper authorization is illegal in most jurisdictions. You are solely responsible for licenses, SAS grants, indoor restrictions, and type approval. HyperonX and contributors provide software as-is for lawful private networks, research, and shared-spectrum regimes only.

## Why mesh at all

- **Geographic extension**: one logical private network across buildings or sites.
- **Resilience**: if a node's uplink dies, traffic can traverse surviving peers (failover; local breakout remains the fallback).
- **Shared services**: on-prem servers, NAS, monitoring reachable from any node's UEs.
- **Community pooling**: multiple operators share transport for a common APN set - with explicit policy control.

Fairwave does **not** mesh for commercial roaming: inter-operator settlement/roaming is explicitly future work (see [roaming-future](roaming-future.md)).

## Architecture: split control and data plane

```mermaid
flowchart TB
    subgraph Node A
        CP_A[control plane (mTLS)]
        WG_A[WireGuard data plane]
    end
    subgraph Node B
        CP_B[control plane (mTLS)]
        WG_B[WireGuard data plane]
    end
    RDZ[rendezvous server]
    MDNS[mDNS (LAN)]
    CP_A <-->|mTLS, signed announcements| RDZ
    CP_A <->|mTLS| CP_B
    CP_A <--> MDNS
    CP_B <--> MDNS
    WG_A <-->|WireGuard UDP/51820| WG_B
```

- **Control plane**: node-to-node mTLS (mesh CA issued at join), used for peer discovery, route exchange, health, and configuration. This is where `peer list` data comes from.
- **Data plane**: WireGuard only - UDP, forward secrecy, per-node keypairs, allowed-IP routing.
- **Discovery**: mDNS (`_fairwave._udp.local`) on a shared LAN; signed announces. Optionally a rendezvous server for NAT'd/remote peers.
- **Route exchange**: each node advertises its UE pool (e.g. `10.44.0.0/24`) plus optional site subnets; the agent renders allowed IPs from the merged table.

## Security properties

| Concern | Control |
| --- | --- |
| Peer identity | Mesh CA, node certs, pinning at join |
| Control channel | mTLS, cert rotation, token TTL at join |
| Data channel | WireGuard, 180 s rekey, keepalives 25 s |
| Discovery spoofing | Signed mDNS/rendezvous announces, replay protection |
| Token leaks | One-time bootstrap tokens, 15 m default TTL |
| Eavesdropping | No plaintext control; no subscriber data over mesh unless policy allows |

Details in [mesh runbook](mesh-runbook.md) and [rendezvous](rendezvous.md).

## When NOT to peer

- **You have no service behind the other node.** Mesh adds attack surface; local breakout is simpler and already default.
- **You cannot vouch for the other operator.** Peering accepts their route advertisements and their UEs' traffic rules; policies apply both ways.
- **Regulatory ambiguity.** Routing subscriber traffic across jurisdictions can change your legal exposure (data flows, LI obligations). When in doubt, do not peer.
- **Your node is exposed.** Mesh requires inbound UDP/51820 and mTLS terminations; if the node sits on untrusted transit, prefer rendezvous-only discovery with firewall rules, or no peering.
- **No revocation capability.** If you cannot reach the mesh CA or revoke a peer promptly, a compromised peer stays trusted. That is a reason not to peer, not a reason to skip revocation.

## Failover semantics

- Peers are probed every 25 s (keepalive); a peer is `unreachable` after 180 s.
- Routes from unreachable peers are withdrawn; traffic falls back to the local PGW breakout.
- Re-joining is automatic when the peer returns (control-plane renegotiation).

## Related

- [Two-box peering tutorial](../tutorials/two-box-peering.md) - hands-on.
- [Mesh runbook](mesh-runbook.md) - setup, keys, routes, NAT.
- [Rendezvous server](rendezvous.md) - spec and deployment.
- [Roaming future](roaming-future.md) - what peering is not yet.
- ADR-0004 (WireGuard vs IPsec) - why WireGuard.
