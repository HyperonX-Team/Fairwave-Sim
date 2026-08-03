---
title: FAQ for Carriers
---

# FAQ for Carriers (MNOs)

Written for mobile network operators evaluating Fairwave — as technology, as a non-threat, and as a potential offload or neutral-host partner.

## Q: Is Fairwave a roaming threat?

**No — and the architecture makes that true, not just policy.** There is no SEPP, no GRX/IPX interconnect, no S6a/S8 peering with other PLMNs, and no settlement/charging layer (see [roaming-future](../peering/roaming-future.md)). A Fairwave node serves only its own HSS subscribers on its own PLMN. Nothing in v0.1 or the M0–M6 roadmap lets a Fairwave node attach subscribers of another operator without an explicit hosted-identity arrangement on that operator's own core.

## Q: Could it cause interference with our network?

Only if deployed illegally. Fairwave ships RF-disabled; arming requires the three-gate process (country code + license acknowledgment + allow-list, ADR-0008). A compliant deployment runs in licensed, shared, or experimental spectrum the operator actually holds — at which point interference questions are the operator's, not the software's. The default lab build emits nothing.

## Q: What is the realistic use case for a carrier?

Three honest ones:

1. **Offload / neutral host:** a Fairwave-compatible box serving *your* SIMs — the node hosts your subscribers' identities in a local HSS under an MVNO/MVNE-style contract, with local breakout. This is the supported path, and it is how community networks can interconnect lawfully today.
2. **Enterprise private networks:** your enterprise customers get a private PLMN slice on-site; you get the integration work, they get ownership of the network.
3. **Research/education:** universities and labs running controlled experiments in test spectrum.

What we do *not* propose: Fairwave boxes masquerading as your network, or backhauling your subscribers' traffic over third-party mesh fabric without contract.

## Q: What would it take to interconnect Fairwave boxes lawfully?

See [roaming-future](../peering/roaming-future.md) for the honest list. In brief: contracts and settlement (commercial), SEPP or GRX-certified interconnects (infrastructure), HSS/UDM federation (engineering), LI and data-protection compliance (legal), and real interop testing (process). Fairwave documents the integration surface (Open5GS-standard S6a/API; CDR export schema in M4) so a partner operator can build against it — but we will not ship inter-operator signalling before a licensed partner exists.

## Q: Is the EPC production-grade?

Open5GS is mature open-source EPC software widely deployed in private networks; Fairwave adds the control plane, provisioning, and gating around it. Production hardening is the explicit content of milestones M4–M6 (retention enforcement, HA/SPOF review, security polish). A carrier-grade deployment should treat M6 as the earliest gate, and even then review the [threat model](/design/threat-model.md) with their own security team.

## Q: What about subscriber data protection?

- The core logs no cleartext IMSI; session records use truncated SHA-256 hashes (ADR-0010).
- SIM credentials are vault-encrypted at rest (ADR-0006); nothing is shared with us — there is no telemetry home, no cloud dependency.
- If a carrier uses Fairwave as a vendor, their DPA with the community operator governs; the software's defaults are documented in [privacy](../security/privacy.md).

## Q: Do you plan to compete with MNO services?

No. Fairwave targets the space carriers have no economic reason to serve: sub-building, indoor, low-density private cells, test labs, and community networks. The mesh is confined to one administrative domain; multi-operator settlement is explicitly not on the roadmap without a partner.

## Q: How do we evaluate it safely?

1. Run the [no-RF quickstart](../tutorials/quickstart-no-rf.md) — nothing leaves the laptop.
2. Inspect `GET /v1/tx/arm` to see gate state; confirm the default build cannot emit.
3. Review the ADRs, threat model, and SBOM/cosign verification ([release signing](../security/release-signing.md)).
4. In your lab, run a test PLMN in a shielded or virtual setup before any over-the-air test in licensed spectrum.

## Related

- [Roaming future](../peering/roaming-future.md) · [Regulator FAQ](faq-regulators.md) · [Not Fairwave](not-fairwave.md)
