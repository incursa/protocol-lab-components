# aioquic RFC9220 WebSocket-over-H3 Executor

`aioquic-rfc9220-websocket` packages the aioquic client proof used by the `quic-dotnet` RFC9220 external peer run. It is a client-side test executor for a target HTTP/3 server that exposes the Incursa WebSocket proof endpoint.

## Supported

- Protocol families: `h3`, `websocket`
- Role: client test executor
- Public scenarios/tests: the five established RFC9220 identities plus exact `http3.websocket.rfc9220.fragmented-binary-echo`
- Fragmented diagnostic: one 6000-byte `0xA5` message, SHA-256 `8f8d8f75d55c80475ffb0c12b1ede7083d6df689e8ef04f05176c5050873bfb7`, sent as masked payloads `[1024, 2048, 2928]` with binary/continuation/continuation opcodes and FIN `false/false/true`
- Exact scenario selection is mandatory; adjacent HTTP/1.1 and HTTP/2 identities return `unsupported`, and unknown identities fail closed
- The five core identities require exact `websocket-smoke`: one QUIC connection, one active RFC 9220 stream, one-second warmup, and five-second measured duration.
- The fragmented identity requires exact `diagnostic`: one QUIC connection, eight active RFC 9220 streams, one-second warmup, ten-second measured duration, one-second cooldown, and ten-second operation timeout.
- Package archive SHA-256 values and immutable executor/target image IDs are required admission inputs.

## Known Unsupported

- Server role
- HTTP/1 and HTTP/2 WebSocket upgrade paths
- Raw QUIC transport scenarios outside HTTP/3
- Broad WebSocket suite coverage beyond the proof endpoint behavior

## Pinned Toolchain

- Base image: `python:3.12-slim@sha256:090ba77e2958f6af52a5341f788b50b032dd4ca28377d2893dcf1ecbdfdfe203`
- Python package: `aioquic==1.3.0`
- Component image tag: `incursa-protocol-lab-aioquic-rfc9220-websocket:0.3.1`
- aioquic license text: `third-party/aioquic-LICENSE.txt`
- Package-local trust anchor: `certs/root.pem` for exact `websocket.plab.test` certificate authentication

## Historical Scenario Evidence

The historical external row is not substituted for package-local evidence. Version `0.3.1` records the selected target role explicitly, allowing the same exact client to retain honest `origin-server` or `proxy` evidence.

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
  -t incursa-protocol-lab-aioquic-rfc9220-websocket:0.3.1 `
  ./executors/aioquic-rfc9220-websocket
```

Run against an RFC9220 proof endpoint reachable from Docker:

```powershell
pwsh ./executors/aioquic-rfc9220-websocket/execute.ps1 `
  -ScenarioId http3.websocket.rfc9220.fragmented-binary-echo `
  -SkipBuild `
  -TargetUrl https://host.docker.internal:4435/websocket-proof `
  -TargetImageId sha256:<target-image-id> `
  -ScenarioPackageSha256 <scenario-package-sha256> `
  -ExecutorPackageSha256 <executor-package-sha256> `
  -TargetPackageSha256 <target-package-sha256>
```

Linux/macOS plan-only smoke:

```bash
PLAB_PLAN_ONLY=1 ./executors/aioquic-rfc9220-websocket/execute.sh
```

## Build Package

```powershell
pwsh ./scripts/package/Build-AioquicRfc9220WebSocketPackage.ps1
```
