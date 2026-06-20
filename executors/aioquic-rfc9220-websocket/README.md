# aioquic RFC9220 WebSocket-over-H3 Executor

`aioquic-rfc9220-websocket` packages the aioquic client proof used by the `quic-dotnet` RFC9220 external peer run. It is a client-side test executor for a target HTTP/3 server that exposes the Incursa WebSocket proof endpoint.

## Supported

- Protocol families: `h3`, `websocket`
- Role: client test executor
- Public scenarios/tests: `http3.websocket.rfc9220.extended-connect`, `http3.websocket.rfc9220.control-frames`, `http3.websocket.rfc9220.text-echo`, `http3.websocket.rfc9220.binary-echo`, `http3.websocket.rfc9220.close`
- External proof scope: Extended CONNECT response metadata, server ping/client pong, client ping/server pong, text echo, 6000-byte binary echo, and close echo

## Known Unsupported

- Server role
- HTTP/1 and HTTP/2 WebSocket upgrade paths
- Raw QUIC transport scenarios outside HTTP/3
- Broad WebSocket suite coverage beyond the proof endpoint behavior

## Pinned Toolchain

- Base image: `python:3.12-slim`
- Python package: `aioquic==1.3.0`
- Component image tag: `incursa-protocol-lab-aioquic-rfc9220-websocket:0.1.7`
- Component image ID: `sha256:6f7c8ab6a96d011c2d6eec71e98cde0325eec16dec8546d73cefbf1c9a46e6d4`
- Component repo digest: `incursa-protocol-lab-aioquic-rfc9220-websocket@sha256:6f7c8ab6a96d011c2d6eec71e98cde0325eec16dec8546d73cefbf1c9a46e6d4`
- Source proof image: `incursa-http3-external-interop-aioquic:latest`
- Source proof image ID: `sha256:b022b70e9f351326bf2951c5a56c5a00808b9f0b226a363d8b1b27014352d975`
- Source proof: `C:\src\incursa\quic-dotnet\.artifacts\http3-websocket-external-aioquic\20260619T-rfc9220-aioquic-001\rfc9220-external-aioquic-proof.json`

## Scenario Evidence

| External row | Scenarios | Status | Evidence |
| --- | --- | --- | --- |
| `aioquic-rfc9220-websocket-client__incursa-server` | Extended CONNECT, server ping/client pong, client ping/server pong, text echo, 6000-byte binary echo, close echo | pass | `C:\src\incursa\quic-dotnet\.artifacts\http3-websocket-external-aioquic\20260619T-rfc9220-aioquic-001` |
| server role | all | unsupported | this package is a client-side RFC9220 proof executor |

## Local Smoke

Plan the wrapper command without Docker execution:

```powershell
pwsh ./executors/aioquic-rfc9220-websocket/execute.ps1 -PlanOnly
```

Build the wrapper image:

```powershell
docker build --build-arg AIOQUIC_VERSION=1.3.0 `
  -f ./executors/aioquic-rfc9220-websocket/docker/aioquic-rfc9220-websocket.Dockerfile `
  -t incursa-protocol-lab-aioquic-rfc9220-websocket:0.1.7 `
  ./executors/aioquic-rfc9220-websocket
```

Run against an RFC9220 proof endpoint reachable from Docker:

```powershell
pwsh ./executors/aioquic-rfc9220-websocket/execute.ps1 `
  -SkipBuild `
  -TargetUrl https://host.docker.internal:4435/websocket-proof
```

Linux/macOS plan-only smoke:

```bash
PLAB_PLAN_ONLY=1 ./executors/aioquic-rfc9220-websocket/execute.sh
```

## Build Package

```powershell
pwsh ./scripts/package/Build-AioquicRfc9220WebSocketPackage.ps1
```
