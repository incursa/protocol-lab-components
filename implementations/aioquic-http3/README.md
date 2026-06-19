# aioquic HTTP/3 Implementation Wrapper

`aioquic-http3` packages the aioquic client/server wrapper currently used by the `quic-dotnet` HTTP/3 external interop lane.

## Supported

- Protocol family: `h3`
- Roles: client and server
- Public scenarios: `http3.core.status`, `http3.headers.response-headers-50x32`, `http3.protocol.qpack-repeated-headers`
- Stable external interop scenarios: `get-small`, `not-found`

## Known Unsupported

- HTTP/1 and HTTP/2
- raw QUIC transport scenarios outside HTTP/3
- `get-large` client and server rows are documented skips with aioquic 1.3.0 because response completion stalls were observed.
- `get-empty`, `many-headers`, and `split-data` are not promoted until they have live package-backed peer proof.

## Pinned Toolchain

- Base image: `python:3.12-slim`
- Python package: `aioquic==1.3.0`
- Component image tag: `incursa-protocol-lab-aioquic-http3:0.1.0`
- Component image ID: `sha256:00ca4bff791e5beb205c35e7874c15ad025e67d4f1ded3fcc5b743459e0fc7c6`
- Component repo digest: `incursa-protocol-lab-aioquic-http3@sha256:00ca4bff791e5beb205c35e7874c15ad025e67d4f1ded3fcc5b743459e0fc7c6`
- Source manifest image ID: `sha256:ef138c09ec4cb224ee283f00768eebba6b2d196d9e869809603606ce0d0c0937`
- Local interop image ID observed during this package proof: `sha256:6f84896f71dc47f2c5c842912b99c90e6c101bf187fc7ecd997192e4f8fb8a5e`
- Source evidence: `C:\src\incursa\quic-dotnet\.artifacts\http3-external\20260619T184606Z\peer-tool-manifest.json`

## Scenario Evidence

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
  -t incursa-protocol-lab-aioquic-http3:0.1.0 `
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
