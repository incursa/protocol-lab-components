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
org.protocol-lab.components.scenario.raw-quic-transport.0.1.0.plabpkg
```

## Packaged Scenarios

- `quic.transport.stream-throughput.1mb`
- `quic.transport.latency.echo-1kb`
- `quic.transport.multiplex.100x64kb`
- `quic.transport.stream-limits.100x64kb`
- `quic.transport.payload.large-1mb`
- `quic.transport.duplex-streams`
- `quic.transport.cancellation.reset-stream`

The smoke suite includes only scenarios currently supported by `quic-go-raw-load`. The cancellation manifest is present as an explicit pending lane and is not advertised as supported by the executor package.
