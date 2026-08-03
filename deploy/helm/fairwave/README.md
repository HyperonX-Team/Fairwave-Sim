# Fairwave Helm chart

Deploys the Fairwave control plane (REST API `:8080`) on Kubernetes. The
Open5GS EPC and srsRAN radio are containers inside the chart only when
`open5gs.enabled` / `enb.enabled` are left on - for a hosted core or a
separately managed RAN, disable them and point the control plane at the
external services via values.

## Install

```console
helm repo add fairwave https://charts.fairwave.invalid  # or point at your mirror
helm install fairwave deploy/helm/fairwave --namespace fairwave --create-namespace
```

## Upgrade

```console
helm upgrade fairwave deploy/helm/fairwave --namespace fairwave
```

## Uninstall

```console
helm uninstall fairwave --namespace fairwave
```

## Configuration

| Parameter | Default | Description |
| --- | --- | --- |
| `control.replicas` | `1` | Control-plane replicas (state is on the PVC; keep 1 unless the store is shared) |
| `control.image.repository` | `fairwave/control` | Image repository |
| `control.image.tag` | `0.1.0` | Image tag |
| `control.resources` | `{cpu: 100m/1, mem: 128Mi/512Mi}` | Requests/limits |
| `control.dataStorage` | `1Gi` | PVC size for `/var/lib/fairwave` |
| `control.tokenSecretName` | `""` | Secret with key `token`; empty = no bearer auth (never store tokens in values) |
| `control.config.mode` | `lab` | `lab` (no RF) or `on-air` intent; TX always requires the RF gate |
| `control.config.plmn` | `999/99` | PLMN rendered into the control config |
| `control.config.apns` | `[internet, ims]` | Default APNs |
| `control.config.maxUes` | `128` | Max UEs |
| `control.config.localBreakout` | `true` | Local breakout policy default |
| `open5gs.enabled` | `true` | Deploy Open5GS EPC in-cluster |
| `open5gs.mongoStorage` | `8Gi` | MongoDB PVC size |
| `enb.enabled` | `true` | Deploy srsRAN radio |
| `enb.mode` | `zmq` | `zmq` (lab, virtual RF) or `rf` (hardware - requires the RF gate flow) |
| `networkPolicies.enabled` | `true` | Namespace-scoped network policies |
| `ingress.enabled` | `false` | Expose the API via Ingress |
| `ingress.host` | `fairwave.example.com` | Ingress host |
| `metrics.enabled` | `true` | Scrape annotations + metrics Service |
| `metrics.serviceMonitor.enabled` | `true` | Prometheus Operator ServiceMonitor (CRD must be installed) |

## Security notes

- No secrets are rendered by this chart. The bearer token lives in a Secret
  you create (`kubectl create secret generic fairwave-token --from-file=token=...`)
  referenced by `control.tokenSecretName`.
- `mode: lab` blocks TX transitions; `mode: rf` additionally requires
  `FAIRWAVE_RF_MODE=hardware` plus a mounted acknowledgment file (see
  `deploy/scripts/rf-gate.sh` and `deploy/docker-compose.rf.yml`).
- Health/liveness probes hit `GET /v1/healthz`.
