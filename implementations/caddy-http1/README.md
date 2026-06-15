# Caddy HTTP/1 Implementation

Lane-scoped Caddy HTTP/1.1 implementation package. This package requires a
`caddy` binary on the worker and does not imply Caddy HTTP/2 or HTTP/3 support.

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
