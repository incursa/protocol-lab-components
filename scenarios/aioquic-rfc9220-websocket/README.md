# aioquic RFC9220 WebSocket-over-H3 Scenario Pack

This component packages the scenario and suite declarations used to run the `aioquic-rfc9220-websocket` client proof executor through the live ProtocolLab controller.

The pack is declarative. It does not provide a WebSocket server or the aioquic client container. The suite is intentionally bound to `aioquic-rfc9220-websocket` so package-backed scheduling stays inside the RFC9220 proof lane.

## Package

- Package ID: `org.protocol-lab.components.scenario.http3-websocket-performance`
- Package version: `0.2.2`
- Suite ID: `aioquic-rfc9220-websocket-proof`
- Scenarios: six exact RFC9220 v2 identities, each byte-locked to public commit `8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574`
- Public authority: six scenario files plus `websocket-smoke` and `diagnostic` load profiles are byte-locked to `protocol-lab` commit `8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574`
- Routing: the five core scenarios use `websocket-smoke`; fragmented binary echo uses the separate non-publishable `diagnostic` suite.
- Test executor: `aioquic-rfc9220-websocket`

## Build

```powershell
pwsh ./scripts/package/Build-AioquicRfc9220WebSocketScenarioPackage.ps1
```
