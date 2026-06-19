# quic-go Raw QUIC Load Executor

This component packages the reusable `quic-go-raw-load` ProtocolLab test executor for raw QUIC transport scenarios. It is a client/load-generator package, not an implementation package.

The executor emits `raw-quic-json` metrics to stdout and expects a `quic://host:port/` target discovered from a ProtocolLab adapter-backed implementation.

## Build

Validate manifests:

```powershell
pwsh ./scripts/package/Validate-ProtocolLabComponentManifests.ps1
```

Run the source smoke tests:

```powershell
go -C ./executors/quic-go-raw-load/source test ./cmd/quic-go-raw-load
```

Build a Windows x64 package:

```powershell
pwsh ./scripts/package/Build-QuicGoRawLoadPackage.ps1 -RuntimeIdentifier win-x64
```

Build a Linux x64 package:

```powershell
pwsh ./scripts/package/Build-QuicGoRawLoadPackage.ps1 -RuntimeIdentifier linux-x64
```

The package artifact is written under `artifacts/packages/` as:

```text
org.protocol-lab.components.executor.quic-go-raw-load.0.1.0.<rid>.plabpkg
```

## Local Wrapper

Run from source when no package binary exists:

```powershell
pwsh ./executors/quic-go-raw-load/execute.ps1 --sni localhost --alpn plab-raw-quic --behavior multiplex-streams --stream-type bidirectional --payload-size-bytes 65536 --payload-direction bidirectional --open-pattern concurrent --connections 1 --streams-per-connection 100 --duration 30s quic://127.0.0.1:4433/
```

## Supported Raw QUIC Scenarios

- `quic.transport.stream-throughput.1mb`
- `quic.transport.latency.echo-1kb`
- `quic.transport.multiplex.100x64kb`
- `quic.transport.stream-limits.100x64kb`
- `quic.transport.payload.large-1mb`
- `quic.transport.duplex-streams`

`quic.transport.cancellation.reset-stream` is declared by the scenario pack as pending. The current load executor does not yet drive reset/cancellation classification.
