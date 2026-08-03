# Gemini & Automation Guardrails

This repo is frequently worked on by AI agents (Gemini, OpenCode, Copilot, etc.). The
rules below are mandatory for automated contributions and are enforced in code review:

1. Commit only via **Conventional Commits** (`feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:` ...).
2. Never commit secrets. SIM keys must remain under `sim/test-vectors/` (lab dummies) or be external (HSM/Vault).
3. Do not enable RF-related defaults (`tx_arm`, `enable_tx`, `rf_mode`) in committed configs — lab/zmq only.
4. Keep the legal banner unchanged in docs/index.md and every TX-related doc.
5. Generated files (proto stubs, OpenAPI bundles, coverage) should not be edited by hand; regenerate via `make gen`.
6. Run `make check` (fmt+lint+unit+secrets-scan) before marking work done.
