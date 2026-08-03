# Fairwave Governance

Fairwave is steered by a small group called the **Fairwave Council** and powered by the
wider contributor community.

## Roles

- **Maintainer** — write access, merges PRs, cuts releases. Listed in [.github/CODEOWNERS](.github/CODEOWNERS).
- **Committer** — can triage issues, review PRs (LGTM counts for small changes).
- **Council** — makes scope, license, release, and spectrum-policy decisions. Seeks consensus.
- **Contributor** — you! Open issues, discuss, PRs welcome.
- **Users** — deployers, operators, and researchers. Your needs drive priority.

## Decision Making

1. **Lazy consensus**: if no objections after 7 days, proposal passes.
2. **Council vote** on license changes, security policy, spectrum policy, and major release cuts.
3. **RFC process** for architecture changes (open with `docs/adr/` ADR, discuss in a PR).

## Adding maintainers

Nominated by the Council, must have:
- 6+ months of high-quality contributions
- No spectrum-policy violations in their history
- Two-factor auth + signed commits enabled
- Read and agree to the threat model ([design/threat-model.md](design/threat-model.md))

## Removal

For abusive conduct, spectrum-safety violations, or long inactivity, a Council majority
vote removes maintainers. All votes logged in `.github/` issue with redacted specifics.

## Spectrum policy

Any PR enabling a TX path outside lab-test vectors must include:
- A country and licenses check (not merely a config flag)
- A reference to a real authorization mechanism (SAS client, spectrum permit, or attenuated lab note)
- Code review by at least one Council member

A spectrum-policy violation (intentional unsafe TX, help with IMSI catching, LP bypass)
is grounds for immediate removal from the community.
