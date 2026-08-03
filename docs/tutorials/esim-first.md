# eSIM First: Mint, Serve, Download

This tutorial runs the lab eSIM loop end to end: mint an activation code,
run the SM-DP+ server, and download the profile with the software eUICC.
The same code works against a real phone's LPA once the ASN.1 transport and
carrier applet milestones land (see [eSIM status](../sim-lifecycle/esim.md));
for now the software eUICC is the client.

## 1. Build the CLI

```sh
make build            # produces bin/fairwave
```

## 2. Mint an activation code

```sh
fairwave --data-dir ./data-esim esim issue \
  --imsi 999991234567001 \
  --address fairwave.local:8443 \
  --qr ./activation.png
```

Output:

```
eSIM profile issued for IMSI 999991234567001
  iccid:  89999014438203475010
  code:   LPA:1$fairwave.local:8443$CT47L7JINTA4
  qr:     activation.png
```

`--eid` pins the profile to one eUICC; `--qr` writes the scannable PNG.
The registry (`./data-esim/esim-registry.json`) is created with 0600
permissions - it holds the lab Milenage credentials, so treat it like the
SIM vault.

## 3. Run the SM-DP+ server

```sh
fairwave --data-dir ./data-esim esim serve --addr 127.0.0.1:8443
```

The server exposes the ES9+ endpoints:

```
POST /es9plus/initiateAuthentication
POST /es9plus/authenticateClient
POST /es9plus/getBoundProfilePackage
POST /es9plus/confirmOrder
POST /es9plus/handleNotification
POST /es9plus/cancelSession
```

Lab mode serves plain HTTP; terminate TLS at a reverse proxy or the control
plane before exposing it beyond localhost.

## 4. Download with the software eUICC

The eUICC package is a Go library; the smallest probe is a 20-line program:

```go
e, _ := euicc.New()
p, err := e.Download(ctx, "http://127.0.0.1:8443",
    "LPA:1$fairwave.local:8443$CT47L7JINTA4", nil)
// p.Profile now holds ICCID, IMSI, EF files; profile is installed + enabled.
```

Expect: MAC verification (AES-128-CMAC over the encrypted package) passes,
the payload decrypts, and the profile installs. Tamper with the package and
the download is refused and the session cancelled - the end-to-end tests in
`core/esim/smdp` cover both paths.

## 5. What happens next in the lab

The installed profile's IMSI matches the HSS seed in `hss-init.sh`, so the
subscriber can attach to the lab network exactly like the physical SIM
vectors - through the normal (lab-gated) RAN path. Attach validation with
the profile from the software eUICC is the same `lab-up` + `assert-lab-up`
loop; a physical phone additionally needs the applet milestone.

## Related

- [SIM lifecycle overview](../sim-lifecycle/index.md) ·
  [eSIM status](../sim-lifecycle/esim.md) · [ADR-0013](../adr/0013-esim.md)
