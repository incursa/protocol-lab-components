# nginx HTTP/3 Implementation

`nginx-http3` packages a Docker-backed nginx HTTP/3 server target for ProtocolLab HTTP application comparisons. It is separate from `nginx-http1` so controller inventory can select the exact protocol lane.

## Package

- Package ID: `org.protocol-lab.components.implementation.nginx-http3`
- Package version: `0.1.8`
- Implementation ID: `nginx-http3`
- Public scenarios: `http3.core.status`, `http3.payload.bytes.1kb`, `http3.payload.bytes.64kb`, `http3.headers.response-headers-50x32`, `http3.protocol.qpack-repeated-headers`
- Package-local not-found fixture: `/not-found` returns 404 for external peer smoke.

## Pinned Build

- Wrapper image tag: `incursa-protocol-lab-nginx-http3:0.1.5`
- Base image: `nginx:1.29.0-alpine`
- Base image digest: `nginx@sha256:d67ea0d64d518b1bb04acde3b00f722ac3e9764b3209a9b0a98924ba35e4b779`
- Linux amd64 manifest digest: `nginx@sha256:845b5424415de5f77dd5753cbb7c1be8bd8e44cc81f20f9705783a02f8848317`
- Pinned Alpine package: `openssl=3.5.7-r0`
- Required capability proof: `nginx -V` includes `--with-http_v3_module`.

The wrapper entrypoint and `run.ps1 -ProofOnly` both fail if the selected nginx build does not advertise the HTTP/3 module.

## Supported And Skipped Coverage

| Area | Status | Notes |
| --- | --- | --- |
| `http3.core.status` | supported | `GET /status` returns JSON metadata. |
| `http3.payload.bytes.1kb` | supported | `GET /bytes/1024` returns a deterministic 1 KB body. |
| `http3.payload.bytes.64kb` | supported | `GET /bytes/65536` returns a deterministic 64 KB body. |
| `http3.headers.response-headers-50x32` | supported | `GET /headers/response?count=50&size=32` returns exactly 50 deterministic 32-byte response headers. |
| `http3.protocol.qpack-repeated-headers` | supported | The h3spec/QPACK executor can exercise the HTTP/3 endpoint. |
| `not-found` | supported fixture | `GET /not-found` returns 404; this is not a public package scenario ID. |
| `http3.payload.bytes.1mb` | skipped | Not promoted until package-backed live proof covers the nginx row. |
| Header-heavy/QPACK | unsupported | No deterministic header-heavy fixture is exposed. |
| Upload, streaming, WebSocket-over-H3 | unsupported | No matching endpoints are configured. |
| Raw QUIC | unsupported | nginx is packaged here as an HTTP/3 server target. |

## Local Commands

Plan without building:

```powershell
pwsh ./implementations/nginx-http3/run.ps1 -PlanOnly
```

Build and prove the nginx HTTP/3 module:

```powershell
pwsh ./implementations/nginx-http3/run.ps1 -ProofOnly
```

Run the server:

```powershell
pwsh ./implementations/nginx-http3/run.ps1 -SkipBuild -Port 5446
```

Build the package:

```powershell
pwsh ./scripts/package/Build-NginxHttp3Package.ps1
```

The generated `.plabpkg` is written under `artifacts/packages/` and is not committed.
