---
name: Feature request
about: Propose a capability for Fairwave
title: "[feature] <short summary>"
labels: enhancement
assignees: ""
---

## Problem

What operator/deployment problem does this solve? (Example: "ops cannot see
per-UE throughput from the UI".)

## Proposed behavior

Describe the change: API surface, config keys, UI elements, deployment
artifacts affected.

## Impact

- Components touched: [control plane | agent | CLI | EPC config | RAN | UI | deploy]
- RF implications: [none | gated — describe which gate (spectrum/check,
  tx/arm, band allow-list) applies] — **feature requests involving TX must
  respect the RF gate; proposals that bypass it will be rejected.**

## Alternatives considered

## Notes for reviewers

- Documentation (docs/) update needed: [yes/no]
- Backwards compatibility with existing lab configs: [yes/no]
