# ADR 0004: Peering data plane — WireGuard (not IPsec or GTP-over-IPsec)

- Status: Accepted
- Date: 2026-08-02

## Context

Fairwave's neighborhood mesh must carry traffic between community cells. Needs:

- Encrypted transport over resident-grade ISP links
- Simple identity tied to mTLS/CA identity
- Reasonable NAT traversal (PI hole on CGNAT)
- Works over UDP/TCP (residential ISPs sometimes block UDP)
- High performance on low-power NUCs / ARM

Options: WireGuard (UDP-only, kernel fast), IPsec (IKEv2, strongSwan), OpenVPN (TCP/UDP,
userspace), custom QUIC tunnel.

## Decision

Use **WireGuard** for the data plane. Control plane bootstrap uses mTLS REST; UDP selection
with TCP fallback (via `wstunnel` or user-supplied SSH) handled by `fairwave-agent`.

## Consequences

- ✅ Extremely low CPU + memory footprint on pizza boxes.
- ✅ Audited codebase, widely understood ops story.
- ✅ PSK support for post-quantum hardening (`wg ... preshared-key`)
- ⚠️ UDP blocking on some residential NATs → fallback tunnels for those peers.
- ⚠️ Key rotation must be managed by our control plane, not `wg-quick`.
- ⚠️ GTP-over-S1 handoff across WAN-grade mesh is not implemented yet (future ADR may add GRE/GTP-U bridging for roaming).

## Alternatives considered

- **IPsec/IKEv2** — heavier, but more enterprise-friendly; leave as future but not default.
- **OpenVPN** — TCP fallback is nice, but userspace does not scale to multiple boxes.
- **GTP-based mesh** — tempting for UE handover, but operations visibility and NAT complications are too high in v0.1.
