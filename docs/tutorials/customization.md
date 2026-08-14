---
title: Customization
---

# Customization: PLMN, TAC, APNs, Bands, Spectrum Profiles

This tutorial covers the operator-facing knobs: PLMN, TAC, APNs, bands, and custom spectrum profiles, and how to rebuild the stack after a change.

> Fairwave defaults to lab/no-RF mode. Transmitting on cellular bands without proper authorization is illegal in most jurisdictions. You are solely responsible for licenses, SAS grants, indoor restrictions, and type approval. HyperonX and contributors provide software as-is for lawful private networks, research, and shared-spectrum regimes only.

All configuration lives in YAML validated by JSON Schema - see [ADR-0012](../adr/0012-config-format.md). The places that matter:

- `deploy/config/fairwave-control.yaml` - node/control-plane configuration used in lab mode (mounted read-only into the container).
- Production deployments use the same schema via `deploy/helm/fairwave/` (ConfigMap) or `deploy/ansible/`.

Environment variables override any key (prefix `FW_`, nested keys separated by `_`), so containers can be parameterized without editing files.

## Changing the PLMN

MCC/MNC are set in the `network` section:

```yaml
network:
  plmn:
    mcc: 999
    mnc: "99"      # quoted: may have a leading zero
  tac: 7
```

Apply and restart:

```bash
make lab-up           # recreates stack with new config
make status
```

```
STATE: on-air
PLMN: 313-100   TAC: 7     # after editing to mcc 313, mnc 100
```

Notes:

- IMSIs minted by the provisioner are prefixed with the operator's MCC/MNC; existing SIMs will be rejected by the HSS after a PLMN change unless re-issued. The provisioner refuses to mix IMSI prefixes in one output run.
- Your IMSI prefix does **not** have to match the served PLMN, but handsets expect the network PLMN to be broadcast; keep TAC inside the serving network's range.

## Changing the TAC

```yaml
network:
  tac: 42
```

Rebuild as above. `fairwave node status` will show the new TAC. Tracking-area updates (TAU) are visible in Open5GS MME logs.

## Changing APNs

```yaml
network:
  apns:
    - name: internet
      type: ipv4
      dns: [1.1.1.1, 8.8.8.8]
    - name: ims
      type: ipv4
      dns: [1.1.1.1]
```

APN names must match the APN configured on the SIM profile and on the UE. When you change APNs, re-issue SIMs whose profile references the old APN (see [provisioner](../sim-lifecycle/provisioner.md)).

## Adding bands and EARFCNs

Radio parameters are in the `radio` section, consumed by srsRAN at build/run time:

```yaml
radio:
  bands:
    - earfcn: 2600          # LTE band 7 downlink EARFCN
      bandwidth_mhz: 10
      dl_earfcn: 2600
      ul_earfcn: 21350
    - earfcn: 3450          # another carrier, same band or not
```

> Adding an EARFCN only configures the stack; it does **not** authorize transmission. Real RF additionally requires the frequency to be inside the armed allow-list - see [spectrum gate ADR](../adr/0008-spectrum-gate.md) and the [spectrum matrix](/design/spectrum-matrix.md).

## Custom spectrum profile YAML

A spectrum profile is a validated YAML document that the control plane checks against before arming TX:

```yaml
# spectrum-profile.yaml
profile:
  id: us-gaa-b48
  description: CBRS GAA band 48 (3550-3700 MHz), 20 MHz TDD
  country: US
  band: 48
  type: shared       # gaa | paa | unlicensed | experimental
  channels:
    - earfcn: 55090
      bandwidth_mhz: 20
      downlink_low_mhz: 3550
      uplink_low_mhz: 3550
      tdd: true
  requirements:
    - sas_grant: optional     # mandated by local rules where applicable
    - indoor_restrictions: true
    - eirp_limit_dbm: 30
```

Install it:

```bash
fairwave spectrum check --profile ./spectrum-profile.yaml
```

```
profile us-gaa-b48: 1 channel
  earfcn 55090   20 MHz   valid for country US
  WARN: no SAS grant attached (GAA); confirm local rules before arming
```

## Rebuilding

Config schema changes are rare (ADR-0012 keeps them backward compatible); changing *values* requires no rebuild, just a restart:

```bash
make lab-up            # recreate with new values
make status            # confirm
fairwave doctor        # full sanity pass
```

If you changed YAML schema version or srsRAN/Open5GS image tags, a full rebuild is:

```bash
./scripts/bootstrap.sh
make build
make lab-up
```

## Checklist for any change

1. Edit YAML (or set `FW_*` env vars).
2. Validate: `fairwave doctor` catches schema errors before apply.
3. Restart: `make lab-up`.
4. Confirm: `make status` reflects the change.
5. Re-issue SIMs if PLMN or APN changed.
6. If the change touches real RF: re-run `fairwave spectrum check` and re-arm TX (the gate is re-evaluated on every config reload).
