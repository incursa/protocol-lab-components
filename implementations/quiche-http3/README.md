# quiche HTTP/3 Implementation Wrapper

`quiche-http3` packages the Docker-backed quiche client/server image used by the `quic-dotnet` HTTP/3 external interop lane.

## Supported

- Protocol family: `h3`
- Roles: client and server
- Public scenarios: `http3.core.status`, `http3.payload.bytes.64kb`, `http3.payload.bytes.1mb`
- Stable external interop scenarios: `get-small`, `get-empty`, `get-large`, `not-found`

## Pinned Peer Image

- Image: `cloudflare/quiche:latest`
- Image ID: `sha256:9f53591499834ffd0d74eae3a67baafec3f9233725cc565852ca13139bdf3b8c`
- Repo digest: `cloudflare/quiche@sha256:9f53591499834ffd0d74eae3a67baafec3f9233725cc565852ca13139bdf3b8c`
- Source evidence: `C:\shared\src\incursa\quic-dotnet\.artifacts\http3-external\20260619T184606Z\peer-tool-manifest.json`

## Local Smoke

Plan the client wrapper command:

```powershell
pwsh ./implementations/quiche-http3/run.ps1 -Mode Client -PlanOnly
```

Run the client against an HTTP/3 target reachable from Docker:

```powershell
pwsh ./implementations/quiche-http3/run.ps1 `
  -Mode Client `
  -Url https://host.docker.internal:4433/small.txt `
  -ConnectTo host.docker.internal:4433
```

## Build Package

```powershell
pwsh ./scripts/package/Build-QuicheHttp3Package.ps1
```
