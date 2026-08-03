## What does this PR do?

<!-- One paragraph. Reference the issue: Fixes #123. -->

## Type of change

- [ ] Bug fix
- [ ] Feature
- [ ] Deploy/infra asset (compose, helm, ansible, terraform)
- [ ] Docs (docs/ only - see note below)
- [ ] Refactor / chore

## Testing

- [ ] `go vet ./...` / lint pass (Go changes)
- [ ] `docker compose -f deploy/docker-compose.yml up -d --build` works (deploy changes)
- [ ] `helm lint deploy/helm/fairwave` passes (helm changes)
- [ ] UI: opened the single-file dashboard, no console errors
- [ ] Manual test described: <!-- what did you run and what did you observe -->

## Security checklist

**This checklist is mandatory. PRs with unchecked security items are not merged.**

- [ ] No secrets committed: no keys, tokens, Ki/OPc, wg private keys, ack
      files, or license references anywhere in the diff (test vectors in
      `sim/test-vectors/` are the only allowed exception - dummy, lab-only)
- [ ] No new paths allow RF TX without the gate: any change touching
      `core/ran/`, `deploy/docker-compose.rf.yml`, `deploy/scripts/rf-gate.sh`
      or `tx/arm` logic preserves the RF gate (country + license
      acknowledgment + band allow-list) - proposals that weaken it are
      rejected
- [ ] Docs updated (docs/ + README where behavior changed)
- [ ] Tests run (unit + the integration smoke described above)
- [ ] `.gitignore` / `.dockerignore` updated for any new artifact type

## Notes for reviewers

<!-- Anything unusual, follow-ups, related PRs. -->
