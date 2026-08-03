---
title: SIM Card Bureau Runbook
---

# SIM Card Bureau Runbook

This runbook covers ordering physical SIM cards from a personalization bureau: what to send, what stays home, how to structure IMSI ranges, transport, and reconciliation.

## What the bureau gets: the encrypted bundle, never raw Ki

The bureau needs (IMSI, ICCID, Ki, OPc) to personalize cards. Fairwave's rule: **Ki and OPc never leave your house in plaintext**.

- Send: `sims-bundle.enc` (AES-256-GCM) + `manifest.txt` + an envelope key delivered over a separate channel (e.g. PGP or an out-of-band key ceremony).
- Keep home: the vault, the cluster KEK, `hss-hook.sh`, and any JSON containing hashes (the bureau does not need hashes).

If your bureau requires CSV, generate `bureau-ready.csv` with the manifest's decryption helper - Ki/OPc are decrypted only inside the transport you and the bureau agreed on (encrypted SFTP / PGP-encrypted file). Never email raw CSV with Ki.

```
home                            bureau
vault (KEK) --decrypt--> bundle.enc --PGP/SSH--> personalization CSV -> cards
```

## What stays home

| Stays home | Reason |
| --- | --- |
| Cluster KEK | Decrypts everything |
| `sims.json`/`sims.csv` | Hashes and metadata only - policy, not secrecy |
| `hss-hook.sh` | Loads into *your* HSS/UDM |
| Audit log | Your compliance record |

## Card profile ordering: IMSI ranges

Structure orders so ranges are self-describing and auditable:

- Reserve one prefix per order lot: `999991200xxxxx`, `999991201xxxxx`, ...
- Keep lab and prod prefixes disjoint (provisioner enforces profile-prefix mapping).
- Block sizes: bureau minimums apply (often 1k–10k per order); order in the granularity the bureau supports, but the provisioner can split one run across multiple IMSI ranges only if they share a profile.
- Record the mapping `order # <-> prefix range <-> profile` in the vault manifest before shipping.

```yaml
# order manifest excerpt (manifest.txt)
order: FW-2026-0802-01
profile: prod
prefixes:
  - range: 9999912xxxxxx
    count: 1000
bureau_contact: personalizer@example.com
transport: encrypted-sftp
```

## Logistics

1. Generate: `fairwave sim issue --profile prod --count 1000 --prefix 9999912`.
2. Review: `manifest.txt`, spot-check hashes against `sims.csv`.
3. Package: `bundle.enc` + manifest; encrypt envelope key separately.
4. Ship: encrypted SFTP or GPG-encrypted attachment - both sides sign manifests.
5. Acknowledge: bureau returns a signed manifest of received files.

Timeline guidance: personalization lead time is bureau-dependent (days to weeks); the provisioner's expiry clock starts at issuance, so order with expiry ≥ personalization + logistics + 30 days buffer.

## Secure transport rules

- No plaintext Ki in email, chat, or ticket systems - ever.
- Two-channel rule: file on channel A, envelope key on channel B.
- Digital signatures on manifests both directions; reconcile hashes.
- If a transport was exposed (breach, wrong recipient), revoke the affected IMSI range *before* cards leave the bureau: `fairwave sim revoke --prefix 9999912 --reason transport-exposure` (see [revocation](revocation.md)).

## Reconciliation

When cards arrive:

1. Spot-check: decode 10–20 cards with any reader; compare (IMSI, ICCID, Ki) against decrypted bundle values.
2. Hash match: ensure `sha256(Ki)` truncated 12 hex equals the value recorded home-side.
3. Load: run `hss-hook.sh` against the HSS/UDM; hook prints `present / dupes / conflicts`.
4. Verify first attach with a physical handset (see [sim-issue-first tutorial](../tutorials/sim-issue-first.md)).
5. Close the order: mark `reconciled` in the manifest; discrepancy rate > 0.5% triggers a bureau review.

## Failure handling

| Event | Action |
| --- | --- |
| Bundle lost in transit | Re-issue the same range (provisioner supports deterministic re-issue from vault) |
| Envelope key compromised | Rotate bundle key, re-encrypt, revoke anything already decrypted |
| Bureau over-ships IMSIs | Revoke surplus range; do not accept partial personalizations into prod |
| Card defect rate high | Retain defective ICCIDs, revoke, claim under SLA |
| Range collision with another operator | Never happens with 15-digit IMSI + your prefix; verify prefix ownership paperwork instead |

## Related

- [Provisioner](provisioner.md) · [Revocation](revocation.md) · [SIM lifecycle](index.md) · [ADR-0006 SIM vault](../adr/0006-sim-vault.md)
