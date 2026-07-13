# aioquic RFC9220 WebSocket-over-H3 Executor

`aioquic-rfc9220-websocket` packages the aioquic client proof used by the `quic-dotnet` RFC9220 external peer run. It is a client-side test executor for a target HTTP/3 server that exposes the Incursa WebSocket proof endpoint.

## Supported

- Protocol families: `h3`, `websocket`
- Role: client test executor
- Public scenarios/tests: the five established RFC9220 identities plus exact `http3.websocket.rfc9220.fragmented-binary-echo`
- Fragmented diagnostic: one 6000-byte `0xA5` message, SHA-256 `8f8d8f75d55c80475ffb0c12b1ede7083d6df689e8ef04f05176c5050873bfb7`, sent as masked payloads `[1024, 2048, 2928]` with binary/continuation/continuation opcodes and FIN `false/false/true`
- Exact scenario selection is mandatory; adjacent HTTP/1.1 and HTTP/2 identities return `unsupported`, and unknown identities fail closed

## Known Unsupported

- Server role
- HTTP/1 and HTTP/2 WebSocket upgrade paths
- Raw QUIC transport scenarios outside HTTP/3
- Broad WebSocket suite coverage beyond the proof endpoint behavior

## Pinned Toolchain

- Base image: `python:3.12-slim@sha256:090ba77e2958f6af52a5341f788b50b032dd4ca28377d2893dcf1ecbdfdfe203`
- Python package: `aioquic==1.3.0`
- Component image tag: `incursa-protocol-lab-aioquic-rfc9220-websocket:0.2.0`
- aioquic license text: `third-party/aioquic-LICENSE.txt`
- Source proof image: `incursa-http3-external-interop-aioquic:latest`
- Source proof image ID: `sha256:b022b70e9f351326bf2951c5a56c5a00808b9f0b226a363d8b1b27014352d975`
- Source proof: `C:\src\incursa\quic-dotnet\.artifacts\http3-websocket-external-aioquic\20260619T-rfc9220-aioquic-001\rfc9220-external-aioquic-proof.json`

## Historical Scenario Evidence

The row below is the original `0.1.7` external proof. It is not substituted for the package-local six-cell `0.2.0` evidence.

| External row | Scenarios | Status | Evidence |
| --- | --- | --- | --- |
| `aioquic-rfc9220-websocket-client__incursa-server` | Extended CONNECT, server ping/client pong, client ping/server pong, text echo, 6000-byte binary echo, close echo | pass | `C:\src\incursa\quic-dotnet\.artifacts\http3-websocket-external-aioquic\20260619T-rfc9220-aioquic-001` |
| server role | all | unsupported | this package is a client-side RFC9220 proof executor |

## Local Smoke

Plan the wrapper command without Docker execution:

```powershell
pwsh ./executors/aioquic-rfc9220-websocket/execute.ps1 -ScenarioId http3.websocket.rfc9220.fragmented-binary-echo -PlanOnly
```

Build the wrapper image:

```powershell
docker build --build-arg AIOQUIC_VERSION=1.3.0 `
  -f ./executors/aioquic-rfc9220-websocket/docker/aioquic-rfc9220-websocket.Dockerfile `
  -t incursa-protocol-lab-aioquic-rfc9220-websocket:0.2.0 `
  ./executors/aioquic-rfc9220-websocket
```

Run against an RFC9220 proof endpoint reachable from Docker:

```powershell
pwsh ./executors/aioquic-rfc9220-websocket/execute.ps1 `
  -ScenarioId http3.websocket.rfc9220.fragmented-binary-echo `
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
