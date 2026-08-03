# Captive portal (onboarding page)

`index.html` - a static, single-file onboarding page shown to devices that
join the network's Wi-Fi before they have a data session (captive detection
redirect). It is served by the control plane (or the site's router) at the
captive portal URL; no build step, no frameworks.

## What it does

- Fetches `/v1/status` (unauthenticated) to display a live network state
  dot - degrades gracefully when the API is unreachable.
- Presents Wi-Fi-calling onboarding instructions (placeholder flow: the
  actual per-carrier instructions are injected by the operator - see
  `docs/software/captive-portal.md`).
- Includes a privacy notice and links back to the Fairwave landing page.

## Serving & security

- Serve over HTTPS when possible; captive portals commonly cannot use a
  pinned cert, so the control plane serves it over HTTP **only** on the
  local network with a strict CSP (`default-src 'none'; script-src 'self';
  connect-src 'self'; style-src 'self'; img-src 'self' data:; base-uri
  'none'; frame-ancestors 'none'`). The page contains no external requests.
- The status fetch is read-only and contains no subscriber data (hashes
  only) - safe for unauthenticated display.
- Never place tokens, keys, or operator credentials in this file.
