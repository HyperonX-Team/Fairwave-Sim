---
title: What Fairwave is NOT
---

# What Fairwave is NOT

Clear boundaries beat clever disclaimers. If a promise about Fairwave does not appear here or in the docs, treat it as unfulfilled.

## Not an IMSI catcher

Fairwave runs a real, identifiable network (its own PLMN), serves only subscribers it provisioned itself, and has no passive identification, impersonation, or device-camping capability. It cannot harvest IMSIs from bystanders; it has no reason to and no code for it. Any fork that adds such capability is not Fairwave, and we do not support it. See the [regulator FAQ](faq-regulators.md).

## Not a free nationwide MNO

Fairwave does not make you a mobile operator, give you spectrum, or interconnect you with anyone. It is a private small-cell toolkit. No SEPP/IPX, no roaming, no settlement — and no road map to change that without a licensed partner ([roaming-future](../peering/roaming-future.md)).

## Not a spectrum free-for-all

RF is disabled by default and gated behind three independent layers (ADR-0008). Fairwave does not grant spectrum rights, does not waive licenses, and does not evaluate your jurisdiction. Transmitting without authorization is illegal and is solely your responsibility.

## Not a law-enforcement evasion tool

There is no anonymity layer, no "unfindable network" feature, and no anti-LI capability. On the contrary: nodes are identifiable by design, the audit log is append-only, and lawful interception, where law requires it, is an operator-side integration we document rather than dodge. If a deployment exists to evade lawful authority, it is misusing the project.

## Not a replacement for 911/112

A private network is not an emergency network. Handsets on a Fairwave cell have no guaranteed emergency routing; public-safety calls must fall back to the public network. Never deploy a cell where it could interfere with public-safety communications, and never present Fairwave as a life-safety service.

## Not "supported as a carrier"

The project is open-source software maintained by HyperonX and contributors. There is no carrier-grade SLA, no 24/7 support desk, and no regulatory representation. Commercial deployments are the responsibility of whoever operates them; M6-hardened releases narrow the risk but do not transfer it.

## Not a black box

Everything is inspectable: code, configs, ADRs, SBOMs, runtime gate state (`/v1/tx/arm`), and audit logs. A "we trust it because it's proprietary" stance is impossible, and we intend it that way.

## The one-sentence version

> Fairwave is a lawful, private, inspectable small-cell toolkit for people who hold or lawfully access spectrum — not a surveillance device, not an MNO replacement, and not a shortcut past regulation.

## Related

- [Regulator FAQ](faq-regulators.md) · [Carrier FAQ](faq-carriers.md) · [Security overview](../security/index.md)
