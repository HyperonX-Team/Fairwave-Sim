---
title: Privacy and Data Minimization
---

# Privacy and Data Minimization

Fairwave's privacy stance: **identify the minimum data the network needs, keep only that, and hash what must be referenced.** ADR-0010 is the binding decision; this page explains the operational effect.

## What we log (and what we refuse to)

| Data | Logged? | Form |
| --- | --- | --- |
| IMSI | No (never in cleartext logs) | `sha256(imsi)` truncated 12 hex |
| Ki / OPc | Never | absent entirely |
| ICCID | Rarely | hashed, or withheld |
| UE IP | Yes (operationally necessary) | cleartext in session records |
| APN, session timestamps | Yes | cleartext (needed for ops) |
| Attach/reject events | Yes | with IMSI hash only |
| Location (cell, TAC) | Yes | coarse: node/cell level, not GPS |
| Full subscriber payload | No | the core does not inspect payloads |

### The hash scheme

- `hash(imsi) = first 12 hex chars of SHA-256(imsi)` — lowercase, e.g. `9f2c41b07d3a`.
- Deliberately truncated: 48 bits of entropy is enough to correlate an operator's own records and *deliberately not* enough to brute-force full IMSI space comfortably at scale, while keeping logs usable. Do not compute it on your own IMSI lists and publish the mapping — that defeats the purpose.
- Applies to session records, dashboards, audit log, metrics labels.

## Dashboards

- Session pages show: hash, APN, IP, timestamps, state — no IMSI, no ICCID.
- The SIM inventory page shows hashes + operator `label` metadata, not credentials.
- Default UI hides revoke reasons after 30 days (retention, below).
- `/metrics` exposes counts by state/APN/PLMN only; no per-subscriber identifiers (see [REST reference](../api/rest.md#metrics)).

## Data retention

| Dataset | Default retention | Notes |
| --- | --- | --- |
| Session records | 30 days | Configurable; audit keeps summary hashes |
| Audit log | 400 days | Append-only, hash-chained daily |
| SIM metadata (hashes) | Until wipe | Wipe removes from all stores |
| Metrics | 90 days (Prometheus retention in lab) | Scale configurable |
| Raw HSS/UDM subscriber records | Until revoked/wiped | Needed for auth |

All retention is operator-configurable via `/v1/policy` and documented in `/design/roadmap.md` (M4: retention policy enforcement job).

## What this means for community operators (DPA-style notes)

These are notes, not legal advice; a real data-processing agreement (DPA) needs counsel.

1. **You are the data controller** for subscriber data in your network. Fairwave is the processor tool. If you host subscriber data on behalf of others (e.g. a community serving members), map who is controller and who is processor *before* launch.
2. **Subscriber consent/notices:** if you operate in GDPR or similar regimes, members must be informed about what is collected (attaches, sessions, IPs) and retention periods. The captive portal is the natural place for a privacy notice (see ADR-0011).
3. **Cross-border peering:** peering moves traffic and metadata across administrative boundaries (see [peering](../peering/index.md)); document the flows.
4. **Data subject rights:** you must be able to delete an individual's records. Fairwave supports this: revoke the SIM, wipe hashes from stores, and purge session records (`fairwave sim wipe --imsi-hash ...`; audit retains only the hash of the action itself).
5. **Sub-processors:** if you use a bureau (see [bureau runbook](../sim-lifecycle/bureau-runbook.md)) or a rendezvous server, list them as sub-processors.
6. **Breach handling:** the audit log and SBOM help you answer "what was exposed". Include the SIM vault in your incident plans: a KEK loss is unrecoverable by design.

## Deliberate limits

- Fairwave cannot prevent you from *adding* IMSI logging in your own forks; we document the default and provide lint (`fairwave doctor --privacy`) that flags cleartext IMSI patterns in config/log sinks.
- Payload inspection is out of scope: the EPC routes, it does not decrypt or DPI. If a deployment needs DPI, that is a deliberate, documented extension — not a hidden feature.

## Related

- ADR-0010 · [Security overview](index.md) · [Regulator FAQ](../reference/faq-regulators.md)
