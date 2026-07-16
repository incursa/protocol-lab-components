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
org.protocol-lab.components.scenario.raw-quic-transport.0.1.16.plabpkg
```

## Packaged Scenarios

- `quic.transport.stream-throughput.64kb`
- `quic.transport.stream-throughput.1mb`
- `quic.transport.stream-download.1mb`
- `quic.transport.stream-throughput.16mb`
- `quic.transport.sustained-stream.256x64kb`
- `quic.transport.sustained-download.256x64kb`
- `quic.transport.latency.echo-1kb`
- `quic.transport.multiplex.100x1kb`
- `quic.transport.multiplex.100x64kb`
- `quic.transport.multiplex.16x1mb`
- `quic.transport.multiplex.mixed-4x16`
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

The package includes both the component smoke suite and the canonical public
`quic-transport-v1-comparison` suite. The cancellation manifest remains an
explicit pending lane. Stream churn remains separate from connection churn and
runs as a stable-connection executor lane. Resumption-rejected,
resumed-handshake, zero-rtt-accepted, and zero-rtt-rejected are packaged here
without claiming executor support yet.

Version `0.1.16` adds exact mixed-size multiplexing across multiple stable connections.
Version `0.1.15` adds the canonical public raw QUIC comparison suite to the
package so every supported peer lane has a selectable suite.
Version `0.1.14` adds exact sustained 256x64KiB server-to-client download coverage.
Version `0.1.13` added exact sustained 256x64KiB client-to-server upload coverage.
Version `0.1.12` added exact deterministic server-to-client download coverage
to the existing orthogonal upload, multiplexing, and duplex evidence.
