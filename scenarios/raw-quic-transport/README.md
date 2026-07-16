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
org.protocol-lab.components.scenario.raw-quic-transport.0.1.12.plabpkg
```

## Packaged Scenarios

- `quic.transport.stream-throughput.64kb`
- `quic.transport.stream-throughput.1mb`
- `quic.transport.stream-download.1mb`
- `quic.transport.stream-throughput.16mb`
- `quic.transport.latency.echo-1kb`
- `quic.transport.multiplex.100x1kb`
- `quic.transport.multiplex.100x64kb`
- `quic.transport.multiplex.16x1mb`
- `quic.transport.stream-limits.100x64kb`
- `quic.transport.flow-control.slow-reader-16x64kb`
- `quic.transport.payload.large-1mb`
- `quic.transport.duplex-streams`
- `quic.transport.duplex-streams.16x1mb`
- `quic.transport.duplex-streams-peer-matrix`
- `quic.transport.cancellation.reset-stream`
- `quic.transport.handshake-cold`
- `quic.transport.connection-churn`
- `quic.transport.stream-churn`
- `quic.transport.resumption-rejected`
- `quic.transport.resumed-handshake`
- `quic.transport.zero-rtt-accepted`
- `quic.transport.zero-rtt-rejected`

The smoke suite includes only scenarios currently supported by `quic-go-raw-load`. The cancellation manifest remains an explicit pending lane. Stream churn remains separate from connection churn and now runs as a stable-connection executor lane. Resumption-rejected, resumed-handshake, zero-rtt-accepted, and zero-rtt-rejected are packaged here without claiming executor support yet.

Version `0.1.12` adds exact deterministic server-to-client download coverage
to the existing orthogonal upload, multiplexing, and duplex evidence.
