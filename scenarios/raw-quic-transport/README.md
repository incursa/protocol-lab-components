# Raw QUIC Transport Scenario Pack

This component packages reusable raw QUIC transport scenarios for ProtocolLab. It is a scenario-pack package and does not include an implementation or a load generator.

## Build

Validate manifests:

```powershell
pwsh ./scripts/package/Validate-ProtocolLabComponentManifests.ps1
```

Build the scenario package:

```powershell
pwsh ./scripts/package/Build-RawQuicScenarioPackage.ps1
```

Run the local package file smoke:

```powershell
pwsh ./scenarios/raw-quic-transport/validate.ps1
```

The package artifact is written under `artifacts/packages/` as:

```text
org.protocol-lab.components.scenario.raw-quic-transport.0.1.5.plabpkg
```

## Packaged Scenarios

- `quic.transport.stream-throughput.1mb`
- `quic.transport.latency.echo-1kb`
- `quic.transport.multiplex.100x64kb`
- `quic.transport.stream-limits.100x64kb`
- `quic.transport.payload.large-1mb`
- `quic.transport.duplex-streams`
- `quic.transport.cancellation.reset-stream`
- `quic.transport.handshake-cold`
- `quic.transport.stream-churn`
- `quic.transport.resumption-rejected`
- `quic.transport.resumed-handshake`
- `quic.transport.zero-rtt-accepted`
- `quic.transport.zero-rtt-rejected`

The smoke suite includes only scenarios currently supported by `quic-go-raw-load`. The cancellation manifest remains an explicit pending lane, and the cold-handshake, stream-churn, resumption-rejected, resumed-handshake, zero-rtt-accepted, and zero-rtt-rejected contracts are packaged here without claiming executor support yet.

Version `0.1.5` also carries the canonical proposed RFC 9000 cold-handshake mapping and a
bounded named profile. The mapping uses only `exercises` and `observes`, and the
resulting coverage remains diagnostic-only. Package presence and a successful
handshake do not imply conformance, certification, or a universal requirement pass.
