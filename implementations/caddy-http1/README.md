# Caddy HTTP/1 Implementation

Lane-scoped Caddy HTTP/1.1 implementation package. Version `0.1.1` builds the
target from the official Caddy `2.11.2` Alpine image pinned by OCI index digest;
it does not depend on a worker-installed Caddy binary and does not imply Caddy
HTTP/2 or HTTP/3 support.

## Supported Slice

- Protocol: `h1`
- Scenarios:
  - `http1.core.plaintext`
  - `http1.core.json`
- Known unsupported:
  - `h2`
  - `h3`
  - TLS/HTTPS
  - request-body echo
  - server-sent-events

## Local Smoke

```powershell
pwsh ./implementations/caddy-http1/run.ps1 -Port 8080
Invoke-WebRequest http://127.0.0.1:8080/plaintext
Invoke-WebRequest http://127.0.0.1:8080/json
```

## Build Package

```powershell
pwsh ./scripts/package/Build-CaddyHttp1Package.ps1
```

Pinned runtime image:

```text
caddy@sha256:834468128c7696cec0ceea6172f7d692daf645ae51983ca76e39da54a97c570d
```
