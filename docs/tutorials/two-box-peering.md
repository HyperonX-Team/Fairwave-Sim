---
title: Two-Box Peering
---

# Two-Box Peering: mDNS/rendezvous + WireGuard

> Fairwave defaults to lab/no-RF mode. Transmitting on cellular bands without proper authorization is illegal in most jurisdictions. You are solely responsible for licenses, SAS grants, indoor restrictions, and type approval. HyperonX and contributors provide software as-is for lawful private networks, research, and shared-spectrum regimes only.

This tutorial joins two Fairwave lab stacks into a peer mesh over WireGuard. Both boxes run the ZMQ virtual radio - no RF - so it is safe to run anywhere. The same steps apply to real nodes once RF is gated on.

## Prerequisites

- Two hosts (or two machines on the same LAN) with the [quickstart](quickstart-no-rf.md) lab stack installed.
- Both boxes on the same Layer-2 network (for mDNS), or reachable UDP endpoints (for rendezvous).
- Ports: WireGuard `51820/udp`, control plane `8080/tcp`, mDNS `5353/udp`.

## Step 1: Initialize each node

On each host:

```bash
fairwave node init --name site-a      # on the first box
fairwave node init --name site-b      # on the second box
```

Output (site-a):

```
node site-a initialized
  node id    a1b2c3d4-e5f6-7890-... (uuid)
  role       private-operator
  wg public  MjTz6B9... (derived)
  discovery  mDNS (_fairwave._udp.local) + rendezvous (optional)
```

## Step 2: Generate a bootstrap token and join

On site-a, mint a short-lived join token:

```bash
fairwave node join --generate-token --ttl 15m
```

```
token: fw-join-b5c8...-15m  (expires in 15 minutes)
```

On site-b, join the mesh:

```bash
fairwave node join --token fw-join-b5c8...-15m
```

```
peer site-a (a1b2c3d4...) added
  wg handshake  completed (first packet)
  routes      10.44.0.0/24 <-> 10.45.0.0/24
```

WireGuard keys are generated per node at `node init`; the token authorizes the *control plane* join, and the agent then performs the WireGuard handshake. All control-plane traffic is mTLS; only the data plane uses WireGuard.

## Step 3: Verify the mesh

On either box:

```bash
fairwave peer list
```

```
PEER       NODE ID                        ENDPOINT           STATE   ROUTES
site-a     a1b2c3d4-e5f6-...              wg://site-a:51820  active  [10.44.0.0/24]
site-b     b2c3d4e5-f6a7-...              wg://site-b:51820  active  [10.45.0.0/24]
```

mDNS discovered both peers on the LAN in the `_fairwave._udp.local` service; with no mDNS (e.g. across NAT), both sides use the rendezvous server's signed announcements.

## Step 4: Route exchange and traffic

The control plane exchanges subnet routes over the mTLS channel; the agent installs them into WireGuard's allowed IPs:

- site-a advertises `10.44.0.0/24` (its UE pool behind PGW).
- site-b advertises `10.45.0.0/24`.
- A UE attached at site-a can ping `10.45.0.2` at site-b through the mesh - this is **peered breakout**; without the mesh, traffic breaks out locally at each edge.

```bash
docker compose -f deployments/lab/docker-compose.yml exec ue ping -c 2 10.45.0.2
```

```
64 bytes from 10.45.0.2: icmp_seq=1 ttl=63 time=3.1 ms
```

## Step 5: Failover behaviour

Mesh peers send keepalives every 25 s. When site-b vanishes (host off, wg handshake timeout):

```bash
fairwave peer list
```

```
site-b   ...   state: unreachable (last seen 1m 8s ago)
```

- Data-plane keepalive and handshake-rekey timeout is 180 s.
- Traffic routed toward site-b falls back: if a second path exists (mesh with ≥3 nodes) it is used; otherwise the default EPS bearer at the local PGW takes over (local breakout is always the fallback).
- When site-b returns, the control plane re-negotiates the WireGuard session automatically. No manual intervention required.

## Security defaults that hold

- Control plane: mTLS, node certificates signed by the mesh CA at join time.
- Data plane: WireGuard, forward secrecy, rotating keys.
- mDNS announces are signed; rendezvous updates are signed and rate-limited (see [rendezvous spec](../peering/rendezvous.md)).
- Bootstrap tokens: one-time, TTL-limited (15 minutes in this tutorial), revoked on first use.

## Next steps

- Full mesh semantics: [peering overview](../peering/index.md), [mesh runbook](../peering/mesh-runbook.md).
- Run a rendezvous server for NAT'd nodes: [rendezvous](../peering/rendezvous.md).
- Understand when peering is a bad idea: [when NOT to peer](../peering/index.md#when-not-to-peer).
