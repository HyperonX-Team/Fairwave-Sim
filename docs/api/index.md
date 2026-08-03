---
title: API Overview
---

# API Overview

Fairwave exposes one primary HTTP API (REST, JSON) on the control plane, plus a metrics endpoint. A gRPC surface is planned; see [gRPC note](grpc.md).

## Base URL

- Lab default: `http://127.0.0.1:8080` (control plane container exposes `8080`).
- Production: `https://<node>:8080` with operator-provided TLS; the control plane refuses plaintext when `tls.enabled` is set.
- The OpenAPI specification lives at `api/openapi.yaml` and is the canonical contract; this directory is prose around it.

## Versioning

- API version is path-prefixed: `/v1/...`.
- Breaking changes bump the prefix (`/v2/...`); additive changes (new endpoints, new optional fields) do not.
- Current: **v1**. The control plane reports its own version at `/v1/version` and `/v1/healthz`.

## Authentication

Two supported schemes (see [operator auth](../security/operator-auth.md) for enrollment):

1. **mTLS** - node/CLI certificates signed by the mesh CA. Preferred for CLI and peer traffic.
2. **Bearer token** - short-lived JWT (15 min) obtained after WebAuthn/TOTP login. Required for operator UI sessions.

Unauthenticated requests return `401` with an `error` body. Roles are enforced per endpoint (RBAC; see [operator auth](../security/operator-auth.md#rbac-roles)).

## Error format

Every error response uses one shape:

```json
{
  "error": {
    "code": "sim_not_found",
    "message": "no SIM with imsi 9999912345678901"
  }
}
```

Codes are machine-readable and stable within a major version. HTTP status codes follow RFC 9110: `400` (bad input), `401` (unauthenticated), `403` (forbidden by role), `404` (not found), `409` (conflict), `429` (rate limited), `5xx` (control plane fault).

## Conventions

- Timestamps are RFC 3339 UTC.
- Subscriber identifiers are always the 12-hex truncated SHA-256 hash, never raw IMSI (ADR-0010).
- Pagination: `?limit=&cursor=` on list endpoints; responses include `next_cursor`.
- Idempotency: mutating endpoints accept `Idempotency-Key`; retries with the same key return the original result.

## Endpoint map

| Area | Endpoints |
| --- | --- |
| Health | `GET /v1/healthz` |
| Node | `GET /v1/status`, `GET /v1/version` |
| Nodes | `GET /v1/nodes` |
| Subscribers | `GET|POST /v1/sims` |
| Peering | `GET /v1/peers` |
| Sessions | `GET /v1/sessions` |
| Policy | `GET|PUT /v1/policy` |
| Spectrum | `POST /v1/spectrum/check` |
| TX gate | `GET|POST /v1/tx/arm` |
| Lifecycle | `POST /v1/lifecycle/transition` |
| Metrics | `GET /metrics` (Prometheus) |

Full reference with examples: [REST reference](rest.md).

## gRPC

- Protos: `api/proto/*.proto`; buf-managed codegen in CI.
- Current status: protos are aspirational contracts; REST is canonical for v0.1. gRPC gateway is planned for M1. See [gRPC note](grpc.md).

## Design notes

- OpenAPI file location: `api/openapi.yaml` (root of repo). Validate changes with `make api-lint` (spectral) in CI.
- All mutating endpoints are idempotent where the semantics allow; state transitions are logged in the audit trail.
- Rate limits are per-principal and documented on each endpoint in [rest.md](rest.md).
