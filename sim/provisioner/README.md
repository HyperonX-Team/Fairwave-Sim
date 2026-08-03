# SIM provisioner

The provisioner lives inside `fairwave-cli sim issue` and `sim revoke`. It
is fully offline-first: it mints Ki/OPc, encrypts them under a cluster KEK
(env `FW_SIM_KEK` or HSM path), and emits artifacts that contain hashes and
encrypted bundles — never plaintext keys.

## Usage

```console
# Lab SIMs (test range — the only issuance the lab allows)
fairwave-cli sim issue --profile lab --prefix 9999912 --count 2 --label floor3-lot1

# Production SIMs — rejected unless HSM + license gates are satisfied
fairwave-cli sim issue --profile prod --prefix 310150 --count 100 --label site-a-2026q3
```

Outputs land in `sims/<YYYY-MM-DD>/` (see docs/sim-lifecycle/provisioner.md):

| Artifact | Contents | Ki/OPc? |
| --- | --- | --- |
| `sims.csv` | IMSI, ICCID, profile, APNs, hashes, expiry | No |
| `sims.json` | Same, machine-readable | No |
| `sims-bundle.enc` | Credentials, AES-256-GCM under the KEK | Yes (encrypted) |
| `hss-hook.sh` | Loads SIMs into the lab HSS (lab profile only) | Consumed locally |
| `manifest.txt` | Counts, prefixes, hashes | No |

Hashes are `sha256(value)` truncated to 12 lowercase hex chars, everywhere.

## Bureau CSV format

For SIMs handed to a bureau for personalization, use
`sim/provisioner/bureau_template.csv` as the transport format:

- **No plaintext Ki/OPc in the CSV.** Columns `ki_encrypted` and
  `opc_encrypted` hold KEK-wrapped values (or a reference ID when the bureau
  uses its own HSM import path).
- The bureau must never see the KEK; the control plane unwraps only at the
  destination HSS/UDM.
- `valid_until` is ISO-8601; the provisioner refuses expiry shorter than the
  profile minimum.

## Security rules

1. Run issuance on an air-gapped or operator machine; `sims/` is git-ignored
   (`sim/.gitignore`) and the provisioner refuses to write inside a git
   working tree without `--force-offline` plus a warning.
2. Never commit `sims/`, vault material, KEKs, or plaintext CSV exports.
3. Audit every issue/load: principal, count, prefix, profile, artifact
   hashes — never credential material.
4. Lab vectors (sim/test-vectors/lab-vectors.yaml) are the ONLY material the
   lab stack accepts; the control plane rejects anything else in lab mode.
5. Revocation: `fairwave-cli sim revoke --imsi ...` or
   `POST /v1/sims/{imsi}/revoke`; revocation propagates to the HSS and the
   peer mesh (docs/sim-lifecycle/revocation.md).

Related: docs/sim-lifecycle/bureau-runbook.md, docs/sim-lifecycle/provisioner.md.
