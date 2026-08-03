# Fairwave Threat Model

> [!IMPORTANT]
> **Legal banner** — Fairwave defaults to lab/no-RF mode. Transmitting on cellular bands
> without authorization is illegal in most jurisdictions. This threat model covers lawful
> private LTE deployments only; it explicitly excludes IMSI-catchers, jamming, or bypass
> of lawful interception requirements.

**Version:** 0.1 (lab)  
**Method:** STRIDE + lateral abuse cases  
**Scope:** Fairwave control plane, agent, SIM vault, peering fabric, operator portal, RF path (lab only), metadata logs.

---

## 1. Assets worth protecting

| Asset | Why it matters |
|-------|----------------|
| **Ki/OPc (SIM credentials)** | If leaked, cloned SIMs attach, full operator-grade fraud possible |
| Operator credentials / WebAuthn keys | Account takeover lets attackers issue/revoke SIMs, change TX gates, peer to evil boxes |
| mTLS CA private keys (control plane) | Forged node identity, rogue peer injection |
| WireGuard private keys + traffic | Eavesdropping / MITM on community backbone |
| UE metadata (SUPI/IMSI, session logs) | Mass surveillance; pseudonymity is a Fairwave guarantee |
| Orchestration tokens (Docker socket, k8s admin) | Full host takeover |
| Spectrum-attestation tokens | Prevents illegal transmitter activation |
| Bill-of-materials + commissioning scripts | Supply-chain attacks become invisible compiles |

## 2. Actors

- **External attacker** — without credentials on the public internet / in RF range
- **Malicious insider** — café/enterprise operator with legitimate UI access but wanting unauthorized spectrum or data access
- **Supplier / CI threat** — compromised build, poisoned base image, malicious PR
- **Upstream dependency compromise** — supply-chain attacks against Open5GS/srsRAN images or Go modules
- **Curious/rogue UE** — SIM trying to authenticate wrongly, botnet scanning PLMN for open attach

## 3. Trust boundaries

1. **Public internet** ↔ Fairwave control plane (northbound API)
2. Fairwave control plane ↔ Open5GS (southbound config)
3. Fairwave control plane ↔ srsRAN eNB (southbound supervision)
4. Fairwave box ↔ peer Fairwave boxes (WireGuard/mTLS)
5. Fairwave box ↔ operator laptop (Web UI, captive portal)
6. SIM vault ↔ provisioner (offline batch, one-way to cards)
7. RF transmitter ↔ physical layer
8. Docker daemon ↔ containers (host root control)

## 4. STRIDE with mitigation & status

| Threat | Category | Mitigations | Status |
|--------|----------|-------------|--------|
| CSWSH/HTML injection in operator portal | S/E | CSP, XX- headers, sanitization, course-grained RBAC | Done |
| mTLS/WireGuard man-in-the-middle during provision | S | Bootstrap tokens (5m TTL), mutual TLS with private CA, probability verification on first contact | Done |
| WebAuthn phishing / TOTP rub | S | Passkey-first; backup codes rate-limited | Done |
| Attack on Open5GS API | T | Isolated container, read-only config mounts, no-host networking on lab | Done |
| Malicious config pushes by control plane | T | All southbound submits are templated, diff-validated, rolled-back on failure | Done |
| SIM provisioner abuse (issue w/o billing) | E | Rate limit, audit log, batch signed outputs | Done |
| Operator account escalation (cafe operator for global) | E | RBAC: `viewer`/`operator`/`admin`/root; root only for TX arm | Done |
| Unauthorized RF enable / band hop | E | Country-code allow-list, compile-time RF gates, lab vs. lab+armed separate builds | Done |
| Traffic sniffing over community backbone | I | WireGuard (ChaCha20-Poly1305), PFS on mTLS control, tun routing | Done |
| IMSI leakage into logs | I | Log scrubber hashes IMSI (SHA-256 truncated 12), doc query | Done |
| DoS on attach (mass attaches) | D | SQN throttling, attach flood detection, per-PLMN rate limiting | Partial (lab only) |
| Docker socket exposure | E/D | Rootless docker by default, role-constrained socket to fairwave-control only | Done |
| Supply-chain CI poisoning | S/T | Cosign sign releases, Syft SBOM, GitHub Actions pinning, security events review | Done |
| Operator credential replay | S | Session cookies Plus WebAuthn; no long-lived tokens without re-auth | Done |
| Data residual on deprovisioned SIMs | I | Ki/OPc wiped; HSS UDR rotates keys | Done |

## 5. Lateral abuse cases explicitly out of scope

- **IMSI catching** — we do not build or support passive interrogation; eNodeB only serves provisioned UEs.
- **Roaming spoofing** — SEPP/IPX forged for external carriers is out of scope.
- **Spektrum flooding/jamming** — all TX paths require authorization; jamming signals are not implemented.

## 6. Mitigation details

### 6.1 SIM credential safety

- **At rest:** NaCl secretbox; per-cluster KEK (CLI-provided or Vault-injected).
- **In wire:** batch CSV/JSON written to card-bureau is **one-way**: Ki/OPc never re-enters production API.
- **In logs:** SIM operations log `imsi_sha256_12 = sha256(imsi)[:12]` only.

### 6.2 Control-plane authority model

- No global "god" token; token scopes: `node.enroll`, `sim.issue`, `tx.arm`, `peer.mesh.join`.
- Every API call carries a context deadline; default 5s; max 60s; anything above is audit-logged.
- RBAC on k8s uses least-privilege roles; controller runs as non-root UID 10001.

### 6.3 Peering fabric authn/authz

- mTLS CA per deployment; node identity = certificate SAN `fw-node-<uuid>`.
- WireGuard uses `PresharedKey` on top of Curve25519 handshake for post-quantum resistance.
- Peer admission requires signed "peering intent" from both sides (operator out-of-band approval).

### 6.4 RF gating

- Compile flags: `FAIRWAVE_RF_MODE=none` (default), `lab-zmq`, `hardware`. Specifying `hardware` without a license env var fails.
- Runtime check: `tx_arm` requires country code + frequency allow-list + operator signature.
- Manual attenuator requirement for bench RF is documented in `hw/sdr-notes/`.

## 7. Privacy commitments

- **No IMSI in logs.** Operator dashboard shows hashed IDs with explicit "identify" break-glass flow (audited).
- **No GTP payload inspection.** Bridge traffic is NATted; no DPI hooks exist in the reference implementation.
- **Consent to onboard.** Captive portal is one-way onboarding; it does not probe MAC or cookies before enrollment.

## 8. Residual risk

- **Open5GS/srsRAN security updates must be tracked upstream.** We ship container-pinned versions; operators should review `digest` changes before upgrading RF environments.
- **Hardware supply-chain tampering** (SDR firmware, UEFI) is acknowledged; SBOM helps but cannot eliminate it.
- **Criminals with large budget may still abuse any small-cell.** Fairwave's gates are a strong deterrent, not absolute prevention.

## 9. Reporting

Contact **security@hyperonx.io** for vulnerabilities; do not post public exploits.

---
*Legally note: This document assumes a lawful private LTE or shared-spectrum deployment. Using Fairwave to run unauthorized spectrum access is contrary to this threat model and the project license's intent.*
