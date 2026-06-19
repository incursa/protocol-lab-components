# curl HTTP/3 Client Test Executor

`curl-http3-client` packages the Docker-backed curl client used by the `quic-dotnet` HTTP/3 external interop lane.

## Supported

- Protocol family: `h3`
- Role: client test executor
- Public scenarios/tests: `http3.core.status`, `http3.payload.bytes.64kb`, `http3.payload.bytes.1mb`, `http3.headers.response-headers-50x32`, `http3.protocol.qpack-repeated-headers`
- External interop scenarios: `get-small`, `get-empty`, `get-large`, `not-found`, `many-headers`, `split-data`

## Pinned Peer Image

- Image: `ghcr.io/macbre/curl-http3`
- Image ID: `sha256:c3a360869a4e132180f458f83af2ce67b873b2302739eda27274dad4f62155f8`
- Repo digest: `ghcr.io/macbre/curl-http3@sha256:c3a360869a4e132180f458f83af2ce67b873b2302739eda27274dad4f62155f8`
- Source evidence: `C:\shared\src\incursa\quic-dotnet\.artifacts\http3-external\20260619T184606Z\peer-tool-manifest.json`

## Local Smoke

Plan the wrapper command without contacting a target:

```powershell
pwsh ./executors/curl-http3-client/execute.ps1 -PlanOnly
```

Run against an HTTP/3 target reachable from Docker:

```powershell
pwsh ./executors/curl-http3-client/execute.ps1 `
  -TargetUrl https://host.docker.internal:8443/status `
  -ExpectedStatus 200
```

Linux/macOS plan-only smoke:

```bash
PLAB_PLAN_ONLY=1 ./executors/curl-http3-client/execute.sh
```

## Build Package

```powershell
pwsh ./scripts/package/Build-CurlHttp3ClientPackage.ps1
```
