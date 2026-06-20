# quiche HTTP/3 Implementation Wrapper

`quiche-http3` packages the Docker-backed quiche client/server image used by the `quic-dotnet` HTTP/3 external interop lane.

## Declared Coverage

- Protocol family: `h3`
- Roles: client and server
- Public scenarios declared for runner classification: `http3.core.status`, `http3.payload.bytes.1kb`, `http3.payload.bytes.64kb`, `http3.payload.bytes.1mb`
- Official 1KB row status: validation-failed; package-backed Docker target serves `/bytes/1024` over H3 with 200/1024 bytes, but `quiche-server` omits `content-type`.
- Stable external interop scenarios: `get-small`, `get-empty`, `get-large`, `not-found`

## Pinned Peer Image

- Runner image: `incursa-protocol-lab-quiche-http3:0.1.3`
- Base image: `cloudflare/quiche:latest`
- Base image ID: `sha256:9f53591499834ffd0d74eae3a67baafec3f9233725cc565852ca13139bdf3b8c`
- Base repo digest: `cloudflare/quiche@sha256:9f53591499834ffd0d74eae3a67baafec3f9233725cc565852ca13139bdf3b8c`
- Source evidence: `C:\src\incursa\quic-dotnet\.artifacts\http3-external\20260619T184606Z\peer-tool-manifest.json`

## Scenario Evidence

| External row | Scenarios | Status | Evidence |
| --- | --- | --- | --- |
| `quiche-client__incursa-server` | `get-small`, `not-found`, `get-large`, `many-headers` | pass | `C:\src\incursa\quic-dotnet\.artifacts\http3-external\20260619T125600Z` |
| `quiche-client__incursa-server` | `get-empty`, `split-data` | pass | `C:\src\incursa\quic-dotnet\.artifacts\http3-external\20260619T130112Z` |
| `incursa-client__quiche-server` | `get-empty`, `get-large` | pass | `C:\src\incursa\quic-dotnet\.artifacts\http3-external\20260619T130112Z`, `C:\src\incursa\quic-dotnet\.artifacts\http3-external\20260619T125037Z` |
| `incursa-client__quiche-server` | `get-small`, `not-found` | pass after earlier peer-server exits | latest passing proof exists under `C:\src\incursa\quic-dotnet\.artifacts\http3-external\20260619T124926Z`; earlier blocker was `peer server exited before client run` in `20260619T124803Z` |
| `incursa-client__quiche-server` | `many-headers`, `split-data` | skipped | peer server rows are not wired for these scenarios in the current harness |

## Local Smoke

Plan the client wrapper command:

```powershell
pwsh ./implementations/quiche-http3/run.ps1 -Mode Client -PlanOnly
```

Run the client against an HTTP/3 target reachable from Docker:

```powershell
pwsh ./implementations/quiche-http3/run.ps1 `
  -Mode Client `
  -Url https://host.docker.internal:4433/bytes/1024 `
  -ConnectTo host.docker.internal:4433
```

## Build Package

```powershell
pwsh ./scripts/package/Build-QuicheHttp3Package.ps1
```
