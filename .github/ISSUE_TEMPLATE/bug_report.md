---
name: Bug report
about: Report a defect in Fairwave (control plane, EPC config, RAN, UI, deploy assets)
title: "[bug] <short summary>"
labels: bug
assignees: ""
---

## Summary

One or two sentences. What did you expect, what happened instead?

## Environment

- Deployment: [compose lab | compose.rf | helm | ansible | raw binaries]
- Component: [fairwave-control | agent | CLI | open5gs | srsRAN | UI | website | deploy]
- Version / commit: (`GET /v1/version` output or git rev)
- OS / kernel: 
- RF mode: [lab / zmq | hardware] — **never include license references or
  ack material in public reports**

## Reproduce

1. ...
2. ...
3. ...

## Expected vs actual

- Expected:
- Actual:

## Logs

Attach the relevant section (control-plane log, `docker compose logs enb`,
`srsue` console). Redact IMSIs (hash convention: 12-hex sha256), IPs,
tokens and keys **before** attaching.

## Checklist

- [ ] No secrets, keys, tokens, or ack/authorization material in this report
- [ ] IMSIs redacted (if any appeared in logs)
- [ ] Reproduced on the latest main commit
