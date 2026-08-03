# ADR 0005: Breakout - Edge NAT local-first, hub optional

- Status: Accepted
- Date: 2026-08-02

## Context

When a UE attaches, Fairwave must decide where its traffic exits. Options:

1. **Hub breakout** - All traffic to a central Fairwave hub (privacy, uniformity)
2. **Local EPC breakout** - Traffic exits via the box's Ethernet/SNAT (low latency, distributed)
3. **Hybrid** - peer-selected per UE/APN policy

## Decision

Default to **Edge NAT local-first breakout**:

- The Open5GS UPF/SGW-U inside each box NATs UE traffic to the Internet via the box's WAN.
- Optional `hub breakout` mode steers traffic over the WireGuard route to a Fairwave-operated or community hub.

## Consequences

- [+] There is no mandatory Fairwave-operated backbone; boxes don’t phone home.
- [+] Latency low and local content (mirror server, kiosk, printer, IoT) never leaves the site.
- [+] Legal exposure reduced: traffic stays geographically local unless opted-out.
- [!] UE roaming across boxes without reattach is hard without hub anchoring; unanswered until M6.
- [!] ISP CGNAT may block inbound; `fairwave-agent` needs NAT traversal skill + optional public relays (locked down).

## Alternatives considered

- **Hub-first** - rejected: centralization is contrary to Fairwave's spirit and introduces a single point of failure.
- **Full local-only** - rejected: breaks when users want internet and the café's uplink fails.

The compromise is a box-local NAT that **defaults to WAN egress**, with a WireGuard fallback rule for hub mode.
