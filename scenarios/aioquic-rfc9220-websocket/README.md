# aioquic RFC9220 WebSocket-over-H3 Scenario Pack

This component packages the scenario and suite declarations used to run the `aioquic-rfc9220-websocket` client proof executor through the live ProtocolLab controller.

The pack is declarative. It does not provide a WebSocket server or the aioquic client container. The suite is intentionally bound to `aioquic-rfc9220-websocket` so package-backed scheduling stays inside the RFC9220 proof lane.

## Package

- Package ID: `org.protocol-lab.components.scenario.aioquic-rfc9220-websocket`
- Package version: `0.2.0`
- Suite ID: `aioquic-rfc9220-websocket-proof`
- Scenarios: the five existing exact RFC9220 identities plus `http3.websocket.rfc9220.fragmented-binary-echo`
- Public authority: fragmented scenario bytes are locked to `protocol-lab` commit `8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574`
- Test executor: `aioquic-rfc9220-websocket`

## Build

```powershell
pwsh ./scripts/package/Build-AioquicRfc9220WebSocketScenarioPackage.ps1
```
