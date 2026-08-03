---
title: SIM Provisioner
---

# SIM Provisioner: Offline-First Credential Minting

The provisioner (part of `fairwave-cli sim issue`) generates subscriber credentials without any network dependency. It is a local, deterministic, auditable tool: you tell it how many SIMs and what profile, it writes Ki/OPc material into a protected vault and renders the artifacts the rest of the world needs.

## Inputs

| Input | Example | Meaning |
| --- | --- | --- |
| `--count` | `100` | Number of SIMs to mint |
| `--profile` | `lab` or `prod` | Profile bundle (APNs, lifetimes, IMSI prefix) |
| `--prefix` | `9999912` | IMSI prefix (must be ≤ 15 digits total) |
| `--label` | `floor3-lot1` | Operator metadata, never sent to the EPC |
| (implicit) | — | Cluster KEK (env `FW_SIM_KEK` or HSM path) |

IMSI construction: `prefix + serial` padded to exactly 15 digits. The provisioner refuses a prefix that would overflow 15 digits and refuses to mix prefixes within one run.

## Profiles: lab vs prod

| Property | `lab` | `prod` |
| --- | --- | --- |
| IMSI prefix | `99999...` (test range) | Operator-assigned range |
| Ki/OPc generation | Random per SIM, same crypto | Random per SIM, same crypto |
| APNs | `internet`, `ims` | Configured (defaults `internet`, `ims`) |
| Lifetime | 7 days default | Configured (e.g. 365 days) |
| Store | Lab vault (separate KEK if configured) | Prod vault |
| Attach behavior | Auto-loaded into lab HSS hook | Manual, gated by ops |
| Logging | Hashes only | Hashes only |

Both profiles use identical crypto; the separation is organizational and enforced by the control plane (a lab SIM cannot authenticate against a prod HSS store and vice versa).

## Ki/OPc generation

- Ki: 16 random bytes per SIM (`crypto/rand`), length/format per 3GPP (128-bit AES-128 key material).
- OPc: derived from the operator OP and Ki via the 3GPP OPc computation (AES-128 based), per 3GPP TS 33.102 / 35.206.
- Each run is recorded with its random seed hash for audit; the seed itself is not persisted.

```
Ki  = random(16 bytes)                     # per SIM
OPc = f(operator OP, Ki)                    # AES-128 mix per 3GPP
Ki/OPc written to vault encrypted at rest    # AES-256-GCM, per-cluster KEK
```

## Outputs

Written to `sims/<YYYY-MM-DD>/` (never committed; see [privacy ADR](../adr/0010-privacy-logging.md)):

| Artifact | Contents | Ki/OPc? |
| --- | --- | --- |
| `sims.csv` | IMSI, ICCID, profile, APNs, hashes, expiry | No |
| `sims.json` | Same as CSV, machine-readable | No |
| `sims-bundle.enc` | Full credentials, AES-256-GCM encrypted | Yes (encrypted) |
| `hss-hook.sh` | Loads credentials into Open5GS HSS / UDM | Yes (consumed locally) |
| `manifest.txt` | Counts, prefixes, hashes of artifacts | No |

Hash format everywhere: `sha256(value)` truncated to 12 lowercase hex characters (`9f2c41b07d3a`).

## The HSS hook

`hss-hook.sh` runs against the local Open5GS database (lab) or the UDM API (prod):

```bash
bash sims/2026-08-02/hss-hook.sh
```

```
[open5gs-hss] adding 100 subscribers (profile lab)
[open5gs-hss] 100 present, 0 dupes, 0 conflicts
```

The hook is the only component that decrypts the vault, and only for the exact SIMs in its manifest. After loading, the bundle's one-time decryption flag is set.

## No network required

The provisioner is fully offline-capable: no API calls, no telemetry, no DNS. Network use is limited to the optional HSS hook (loopback) and to operators explicitly pushing bundles to a bureau transport of their choosing. Air-gapped issuance is a first-class scenario.

## Secrets handling

- Full Ki/OPc exist only in the encrypted vault and briefly in process memory during minting/loading.
- Never committed: the provisioner refuses to run if the output directory is inside a git working tree without `--force-offline` and a warning.
- KEK handling: `FW_SIM_KEK` env or HSM slot; KEK never written to any file by the provisioner.
- Audit: every issue/load records principal, count, prefix, profile, artifact hashes — no credential material.

## Related

- [SIM lifecycle overview](index.md) · [Bureau runbook](bureau-runbook.md) · [Revocation](revocation.md) · [ADR-0006 SIM vault](../adr/0006-sim-vault.md) · [ADR-0010 privacy logging](../adr/0010-privacy-logging.md)
