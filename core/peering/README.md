# Peering (WireGuard mesh)

Fairwave nodes form a WireGuard mesh for control traffic and optional
breakout. `wg.example.conf` is the template — **keys are managed by the
fairwave control plane, never committed, never pasted into this repo**.

## Control-plane-managed keys

1. `POST /v1/nodes/{id}/enroll` — node presents its bootstrap token; the
   control plane issues it a WireGuard keypair and a mesh address
   (10.99.0.0/16), then rewrites the node's `wg0.conf` and every peer's
   `AllowedIPs`.
2. `POST /v1/nodes/{id}/leave` — the node's keypair is revoked, its mesh
   address released, and all peers drop it.

Operators must never hand-edit `AllowedIPs` or endpoints: the control plane
reconciles the mesh state and overwrites the config on change. Inspect state
with `wg show` / `GET /v1/peers` — both expose the same truth.

## Local breakout

Default policy is `local_breakout: true` (see `GET/PUT /v1/policy`): UE
traffic exits at the serving node. When a `hub_peer` is set, the hub carries
non-local prefixes advertised via `AllowedIPs`. This is the cloud breakout
deployment shape (`deploy/terraform`).

## Rules

- `wg.example.conf` is the ONLY peering file in this repo. Never rename it
  to `wg0.conf` with real keys.
- Keys live in `/etc/fairwave/env` (hosts, mode 0600) or a Kubernetes Secret
  (`control.tokenSecretName` pattern applies to keys too).
- Rotate a peer by unenrolling and re-enrolling it — never by editing
  `PrivateKey` by hand.
- See docs/peering/mesh-runbook.md and docs/peering/rendezvous.md.
