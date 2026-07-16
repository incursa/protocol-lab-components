# nginx HTTP/1 Implementation

Lane-scoped nginx HTTP/1.1 implementation package. Version `0.1.1` builds the
target from the official nginx `1.29.0` Alpine image pinned by OCI index digest;
it does not depend on a worker-installed nginx binary and does not imply nginx
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
pwsh ./implementations/nginx-http1/run.ps1 -Port 8080
Invoke-WebRequest http://127.0.0.1:8080/plaintext
Invoke-WebRequest http://127.0.0.1:8080/json
```

## Build Package

```powershell
pwsh ./scripts/package/Build-NginxHttp1Package.ps1
```

Pinned runtime image:

```text
nginx@sha256:d67ea0d64d518b1bb04acde3b00f722ac3e9764b3209a9b0a98924ba35e4b779
```
