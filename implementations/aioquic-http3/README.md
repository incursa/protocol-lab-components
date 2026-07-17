# aioquic HTTP/3 Implementation Wrapper

`aioquic-http3` packages an aioquic HTTP/3 origin-server target. Package version `0.3.4` adds the canonical deterministic 64 KiB payload endpoint to the existing origin, QPACK diagnostic, and RFC 9220 WebSocket surfaces.

## Supported

- Protocol family: `h3`
- Comparison role: origin server
- Public scenarios: `http3.core.status`, `http3.payload.bytes.1kb`, `http3.payload.bytes.64kb`, `http3.headers.response-headers-50x32`, `http3.protocol.qpack-repeated-headers`, and all six exact committed RFC9220 WebSocket scenarios including fragmented binary echo.
- Stable external interop scenarios: `get-small`, `not-found`

## Known Unsupported

- HTTP/1 and HTTP/2
- raw QUIC transport scenarios outside HTTP/3
- `get-large` client and server rows are documented skips with aioquic 1.3.0 because response completion stalls were observed.
- `get-empty`, `many-headers`, and `split-data` are not promoted until they have live package-backed peer proof.

## Pinned Toolchain

- Base image: `python:3.12-slim@sha256:090ba77e2958f6af52a5341f788b50b032dd4ca28377d2893dcf1ecbdfdfe203`
- Python package: `aioquic==1.3.0`
- Component image tag: `incursa-protocol-lab-aioquic-http3:0.3.4`
- aioquic license text: `third-party/aioquic-LICENSE.txt`
- Fixed package-local certificate: P-256 self-signed `websocket.plab.test`, authenticated by the executor through its package-local trust copy
- Image identity is recorded from each extracted-package build and must be an immutable `sha256:` ID.

## Historical Scenario Evidence

The rows below predate `0.3.2`; they remain provenance for the unchanged HTTP/3 core behavior, not proof of the current package identity.

| External row | Scenarios | Status | Evidence |
| --- | --- | --- | --- |
| `aioquic-client__incursa-server` | `get-small`, `not-found` | pass | `C:\src\incursa\quic-dotnet\.artifacts\http3-external\20260619T055801Z` |
| `incursa-client__aioquic-server` | `get-small`, `not-found` | pass | `C:\src\incursa\quic-dotnet\.artifacts\http3-external\20260619T055801Z` |
| `aioquic-client__incursa-server`, `incursa-client__aioquic-server` | `get-large` | skipped | aioquic 1.3.0 large-body peer incompatibility; see `C:\src\incursa\quic-dotnet\.artifacts\http3-external\20260619T055646Z` |

## Local Smoke

Plan the client wrapper command without Docker execution:

```powershell
pwsh ./implementations/aioquic-http3/run.ps1 -Mode Client -PlanOnly
```

Build the wrapper image:

```powershell
docker build --build-arg AIOQUIC_VERSION=1.3.0 `
  -f ./implementations/aioquic-http3/docker/aioquic.Dockerfile `
  -t incursa-protocol-lab-aioquic-http3:0.3.4 `
  ./implementations/aioquic-http3
```

Run the client against an HTTP/3 target reachable from Docker:

```powershell
pwsh ./implementations/aioquic-http3/run.ps1 `
  -Mode Client `
  -SkipBuild `
  -Url https://host.docker.internal:8443/status `
  -ExpectedStatus 200
```

Plan-only Linux/macOS smoke:

```bash
PLAB_PLAN_ONLY=1 ./implementations/aioquic-http3/run.sh
```

## Build Package

```powershell
pwsh ./scripts/package/Build-AioquicHttp3Package.ps1
```
