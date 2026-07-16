# quic-go Raw QUIC Load Executor

This component packages the reusable `quic-go-raw-load` ProtocolLab test executor for raw QUIC transport scenarios. It is a client/load-generator package, not an implementation package.

The executor emits a versioned `raw-quic-json` result envelope with metrics,
package-selected executor/load-generator identities, and the
requested/effective load-shape echo. It expects a
`quic://host:port/` target discovered from a ProtocolLab adapter-backed
implementation.

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
org.protocol-lab.components.executor.quic-go-raw-load.0.1.14.<rid>.plabpkg
```

## Local Wrapper

Run from source when no package binary exists:

```powershell
pwsh ./executors/quic-go-raw-load/execute.ps1 --sni localhost --alpn plab-raw-quic --behavior multiplex-streams --stream-type bidirectional --payload-size-bytes 65536 --payload-direction bidirectional --open-pattern concurrent --connections 1 --streams-per-connection 100 --duration 30s quic://127.0.0.1:4433/
```

## Supported Raw QUIC Scenarios

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
- `quic.transport.handshake-cold`
- `quic.transport.connection-churn`
- `quic.transport.stream-churn`
- `quic.transport.duplex-streams`
- `quic.transport.duplex-streams.16x1mb`
- `quic.transport.duplex-streams-peer-matrix`

`quic.transport.cancellation.reset-stream` is declared by the scenario pack as pending. The current load executor does not yet drive reset/cancellation classification.

Version `0.1.14` adds exact round-robin mixed-size multiplexing across stable connections.
Version `0.1.13` fixes the public package declaration for the already-proven
exact-content sustained 256x64KiB server-to-client download lane.
Version `0.1.12` added exact-content sustained 256x64KiB server-to-client downloads.
Version `0.1.11` added exact 256x64KiB sustained writes on one long-lived stream.
Version `0.1.10` added exact-content server-to-client download validation while
keeping the 16-byte request prelude outside payload byte metrics.
