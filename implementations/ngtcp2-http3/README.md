# ngtcp2/nghttp3 HTTP/3 Implementation Wrapper

`ngtcp2-http3` packages the Docker-backed ngtcp2/nghttp3 client/server image used by the `quic-dotnet` HTTP/3 external interop lane.

## Declared Coverage

- Protocol family: `h3`
- Roles: client and server
- Public scenarios declared for runner classification: `http3.external.peer-characterization` marker only; this is not an official benchmark scenario.
- Official ProtocolLab HTTP/3 status and payload row status: validation-failed; package-backed Docker target serves the expected bodies over H3, but `wsslserver` emits `text/plain` instead of the required scenario `content-type` values.
- Stable external interop scenarios: `get-small`, `get-empty`, `get-large`, `not-found`

## Pinned Peer Image

- Runner image: `incursa-protocol-lab-ngtcp2-http3:0.1.2`
- Base image: `ghcr.io/ngtcp2/ngtcp2-interop:latest`
- Base image ID: `sha256:f3703cc822d79f246bb44bbf89b6632438730c52b5c23aaa305c8bbda29f27af`
- Base repo digest: `ghcr.io/ngtcp2/ngtcp2-interop@sha256:f3703cc822d79f246bb44bbf89b6632438730c52b5c23aaa305c8bbda29f27af`
- Source evidence: `C:\src\incursa\quic-dotnet\.artifacts\http3-external\20260619T184606Z\peer-tool-manifest.json`

## Scenario Evidence

| External row | Scenarios | Status | Evidence |
| --- | --- | --- | --- |
| `ngtcp2-client__incursa-server` | `get-small`, `not-found`, `get-large`, `many-headers` | pass | `C:\src\incursa\quic-dotnet\.artifacts\http3-external\20260619T125600Z` |
| `ngtcp2-client__incursa-server` | `get-empty`, `split-data` | pass | `C:\src\incursa\quic-dotnet\.artifacts\http3-external\20260619T130112Z` |
| `incursa-client__ngtcp2-server` | `get-empty`, `get-small`, `not-found`, `get-large` | pass after earlier handshake timeouts | latest passing proofs exist under `C:\src\incursa\quic-dotnet\.artifacts\http3-external\20260619T130437Z` and `C:\src\incursa\quic-dotnet\.artifacts\http3-external\20260619T184808Z`; earlier failures reported `handshake timeout` |
| `incursa-client__ngtcp2-server` | `many-headers`, `split-data` | skipped | peer server rows are not wired for these scenarios in the current harness |

External interop rows remain useful characterization evidence, but they are not accepted official ProtocolLab rows until the server under test emits the required scenario content types.

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
  -Url https://host.docker.internal:4433/bytes/1024
```

## Build Package

```powershell
pwsh ./scripts/package/Build-Ngtcp2Http3Package.ps1
```
