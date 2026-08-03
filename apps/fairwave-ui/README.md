# Fairwave operator UI

Single-file, framework-free dashboard (`index.html`) served statically by
the control plane - no build step, no node_modules.

## Serving

The control plane serves this directory as the static root (config:
`ui_dir`, default `apps/fairwave-ui` or `/usr/share/fairwave-ui` on packaged
nodes). All API calls are relative (`/v1/...`), so the UI works from the
same origin as the API:

```
http://127.0.0.1:8080/            → dashboard
http://127.0.0.1:8080/v1/status   → API
```

Reverse proxies must not rewrite `/v1` paths and must pass the `Authorization`
header through.

## Auth

The dashboard reads the bearer token from `localStorage["fairwave_token"]`.
Use **Set API token** in the toolbar; it prompts once and stores locally.
Tokens never leave the browser except in the `Authorization` header. Clear it
with the same button (empty input). Server-side, auth is optional - with
`tokenSecretName`/`FAIRWAVE_TOKEN` unset the API runs unauthenticated
(lab only).

## CSP

The control plane sends a restrictive Content-Security-Policy on the UI
responses:

```
default-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline';
connect-src 'self'; img-src 'self' data:; base-uri 'none';
frame-ancestors 'none'
```

`index.html` complies: no inline `on*=` handlers, no external resources.
The `style-src 'unsafe-inline'` allowance exists because the stylesheet is a
single `<style>` block in the file; tightening to `style-src 'self'` is a
drop-in once the CSS moves to a small `style.css`.

## Testing

Open the page, enter a token, verify the six status cards + three tables
refresh, and that "Transition to on-air" is refused (or succeeds with the
proper gates) - check the network tab for `{"error":{...}}` bodies, which
the page surfaces in the error banner.
