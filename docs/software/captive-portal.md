---
title: Captive Portal
---

# Captive Portal

The captive portal is the box's front door for devices that are **not** on cellular: Wi-Fi clients, laptops, and phones running Wi-Fi calling (VoWiFi over `ims` APN). It runs on the box; no external service is involved.

## What It Does

- Serves the network's captive page on the box's Wi-Fi SSID / LAN range (standard captive-portal detection: `204`-style redirect / DHCP option 114 where supported).
- **Onboarding for non-cellular devices:** guests accept a brief network policy, choose a network profile if offered, and get a device record (MAC-hashed) with a local IP lease.
- **VoWiFi bootstrap:** for phones with Wi-Fi calling enabled, the portal explains the `ims` APN and shows the ePDG/IMS reachability status; it cannot provision the handset automatically (operator apps are out of scope in v0.1).
- Guest usage is time-boxed and counted per day; counts surface in the operator UI, identities do not.

## Behavior

```mermaid
flowchart LR
    D[Guest device] -->|Wi-Fi| P[Portal]
    P -->|policy accept| R[Local lease / VoWiFi info]
    R --> NET[Internet via local breakout]
```

- First-visit flow: device hits portal → policy page → accept → lease/redirect.
- Returning devices with a stored record skip the accept page.
- The portal never sees cellular IMSIs; it only handles Wi-Fi/device identities.

## Privacy Note

- MAC addresses are stored as truncated sha256 hashes, same rule as IMSIs (`docs/architecture/security.md`).
- Web browsing is NOT inspected. The portal does not MITM TLS; it only serves the landing page and a policy record.
- Guest session logs keep counts and hashes; raw device identifiers are not retained after the session.

## Limits (v0.1)

- No user accounts, no billing, no BYOD/MDM integration.
- VoWiFi is *explained and monitored*, not auto-provisioned.
- Bandwidth shaping is policy-level (per APN), not per-guest QoS yet.

## Related

- Operator UI (operator-facing): `docs/software/operator-ui.md`
- `ims` APN: `docs/architecture/mobile-core.md`
- Security model: `docs/architecture/security.md`
