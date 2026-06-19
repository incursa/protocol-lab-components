# ngtcp2/nghttp3 HTTP/3 Implementation Wrapper

`ngtcp2-http3` packages the Docker-backed ngtcp2/nghttp3 client/server image used by the `quic-dotnet` HTTP/3 external interop lane.

## Supported

- Protocol family: `h3`
- Roles: client and server
- Public scenarios: `http3.core.status`, `http3.payload.bytes.64kb`, `http3.payload.bytes.1mb`
- Stable external interop scenarios: `get-small`, `get-empty`, `get-large`, `not-found`

## Pinned Peer Image

- Image: `ghcr.io/ngtcp2/ngtcp2-interop:latest`
- Image ID: `sha256:f3703cc822d79f246bb44bbf89b6632438730c52b5c23aaa305c8bbda29f27af`
- Repo digest: `ghcr.io/ngtcp2/ngtcp2-interop@sha256:f3703cc822d79f246bb44bbf89b6632438730c52b5c23aaa305c8bbda29f27af`
- Source evidence: `C:\shared\src\incursa\quic-dotnet\.artifacts\http3-external\20260619T184606Z\peer-tool-manifest.json`

## Local Smoke

Plan the client wrapper command:

```powershell
pwsh ./implementations/ngtcp2-http3/run.ps1 -Mode Client -PlanOnly
```

Run the client against an HTTP/3 target reachable from Docker:

```powershell
pwsh ./implementations/ngtcp2-http3/run.ps1 `
  -Mode Client `
  -HostName host.docker.internal `
  -PeerPort 4433 `
  -Url https://host.docker.internal:4433/small.txt
```

## Build Package

```powershell
pwsh ./scripts/package/Build-Ngtcp2Http3Package.ps1
```
