---
title: Backup and Restore
---

# Backup and Restore

A Fairwave box is a state machine: identity, SIMs, policy, peers. Backups protect that state, not the OS (which re-provisions from the golden image, `docs/hardware/image.md`).

## What to Back Up

| Path | Contents | Sensitive |
|---|---|---|
| `/var/lib/fairwave/store/` | nodes, sims, peers, policy, lifecycle JSON | partial (hashes) |
| `/var/lib/fairwave/keys/` | node key, mesh CA material | **yes — private keys** |
| `/var/lib/fairwave/vault/` | Ki/OPc, SIM credential records | **yes — highest sensitivity** |
| `/var/lib/fairwave/open5gs/` + `srsran/` | rendered configs | medium (EIRP, bands, creds in HSS refs) |
| HSS subscriber records | Open5GS HSS database export | yes — SIM material |

Not backed up: container images (re-pulled), OS (re-imaged), caches.

## Backup Procedure (v0.1)

```text
# snapshot data dir with keys + vault
tar -czf fairwave-backup-$(date +%F).tar.gz \
    --exclude='/var/lib/fairwave/logs' \
    /var/lib/fairwave

# HSS export (via open5gs container)
docker compose exec hss open5gs-dbctl export > hss-export.json
```

- Store the archive **off-box** (USB drive, or your own server over the mesh — never plaintext on the open Internet).
- Encrypt at rest: `age`/`gpg` with a key kept separately from the box. The vault is already encrypted with the boot secret; the archive adds a second layer for transport.
- **Frequency:** before any upgrade, after enrollment/peer changes, after SIM issue/revoke batches, before teardown of a pilot (`docs/ops/cafe-pilot.md`).

## Restore Procedure

1. Fresh golden image boot (`docs/hardware/image.md`).
2. Install `fairwave-control`, agent, compose stack.
3. Stop services; extract the archive to `/var/lib/fairwave/` (preserve 0600/0700 modes).
4. Re-enter the vault boot secret.
5. `fairwave node status` → control plane reconciles; verify peers re-establish mTLS (mesh CA intact) — if the archive is from a *different* CA generation, peers must be re-enrolled, not restored.
6. Import HSS export; `fairwave doctor` full pass.
7. Re-arm TX only after the checklist (`docs/spectrum-and-law/compliance-checklist.md`).

## Disaster Notes

- **Lost mesh CA + keys:** the neighborhood must re-generate a CA and re-enroll every peer. IMSI/SIM data survives (vault + HSS); trust does not.
- **Lost vault boot secret:** SIMs become un-programmable (Ki/OPc unrecoverable); re-issue SIMs from the vault backup or re-import from HSS if HSS copy exists.
- **Stolen box:** assume keys compromised; revoke SIMs, drop peers, rotate CA, restore to fresh hardware. See `docs/ops/incident-response.md`.
- **RPO/RTO (honest):** v0.1 has no continuous replication; you lose at most "since last backup" of state, and the mesh regenerates trust only with manual re-enrollment. That is acceptable for a community box, not for a carrier.

## Related

- Security of stored material: `docs/architecture/security.md`
- Incident handling: `docs/ops/incident-response.md`
- Pilot teardown: `docs/ops/cafe-pilot.md`
