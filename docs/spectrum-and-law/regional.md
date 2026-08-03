---
title: Regional Notes
---

# Regional Notes

> Fairwave defaults to lab/no-RF mode. Transmitting on cellular bands without proper authorization is illegal in most jurisdictions. You are solely responsible for licenses, SAS grants, indoor restrictions, and type approval. HyperonX and contributors provide software as-is for lawful private networks, research, and shared-spectrum regimes only.

Region details below are indicative, not legal advice, and change frequently. Verify against the regulator's current rules before any RF work. Per-band engineering data lives in `design/spectrum-matrix.md`.

## Region Table

| Region | Regime | Authority | What you typically need | Fairwave profile |
|---|---|---|---|---|
| US | CBRS GAA/PPA on 3.5 GHz | FCC Part 96 + SAS | certified SAS client, certified radio, grant | `cbrs` |
| US | Other bands | FCC experimental / parts 15, 90, etc. | per-band authorization | `experimental` |
| UK | Shared Access Licence (SAL) | Ofcom | individual licence per site/band, low-power | `community` |
| EU | local/private licenses | national regulators (DE BNetzA, FR ANFR, NL RDI, …) | national local-license application, band/region specific | `community` |
| EU | research/campus | national experimentation rules | campus or experimentation licence | `experimental` |
| India | captive networks | DoT | captive-network registration, no PSTN interworking | `community` (registered) |
| Australia | class/spectrum licences | ACMA | applicable licence; experimental licence for testing | `community` / `experimental` |
| Any | lab/no-RF | none | nothing | `lab` (default) |

## Notes by Region

**US** - Outside CBRS, most cellular bands are assigned to licensees; a small cell needs a real authorization or an FCC experimental licence. CBRS specifics: `docs/spectrum-and-law/cbrs.md`.

**UK** - Ofcom SAL allows low-power individual licensing on certain bands (including parts of 3.8 GHz and others); EIRP and coexistence obligations apply per licence. Indoor/outdoor matters: indoor SAL terms differ.

**EU** - No single "EU licence." Each member state issues local/private 5G/4G licences on distinct bands (e.g. 3.8–4.2 GHz local licences in several states). Type approval and conformity (RED) apply to the radio; your SDR + eNB stack is a transmitter, not a CE-marked terminal in most members' eyes - check.

**Experimental / campus profiles** - For research: confinement to a defined site, low EIRP, registration with the regulator where required, and logs. Fairwave's `experimental` profile encodes: hard EIRP cap, indoor-only flag, site field, mandatory audit logs. None of these substitute for the licence itself.

**India** - Captive (campus) networks under telecom licensing rules: registration, no interconnection to the public switched network, no mobility across the campus boundary. Bands and rules are operator-of-record dependent; contact the DoT/telecom service provider channel for current terms.

**Australia** - ACMA: class licences cover some low-power uses; higher-power or cellular uses need spectrum licences or an experimental licence. ACMA keeps a spectrum licence register you can check for your site.

## What the Profile Actually Does

`fairwave policy set --profile experimental` writes policy fields: `indoor_only: true`, `eirp_cap_dbm`, `site_ref`, `audit_log: mandatory`. The `tx/arm` gate then requires those fields to be populated and refuses to arm otherwise. Again: filling fields is not obtaining authorization.

## Related

- Gate mechanics: `docs/spectrum-and-law/index.md`
- CBRS: `docs/spectrum-and-law/cbrs.md`
- Checklist: `docs/spectrum-and-law/compliance-checklist.md`
- Matrix: `design/spectrum-matrix.md`
