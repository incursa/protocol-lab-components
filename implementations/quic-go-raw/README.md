# quic-go Raw QUIC

`quic-go-raw` packages a quic-go raw QUIC transport target for ProtocolLab raw
QUIC comparisons. It is intentionally separate from the quic-go HTTP/3
implementation and from the `quic-go-raw-load` executor package.

Package ID: `org.protocol-lab.components.implementation.quic-go-raw`

Supported scenarios:

- `quic.transport.stream-throughput.1mb`
- `quic.transport.multiplex.100x64kb`
- `quic.transport.duplex-streams`

Unsupported until proven:

- HTTP/3 scenarios
- raw QUIC latency, stream-limit, large-payload, and cancellation lanes

The package's 64 KiB echo behavior is the duplex lane proof.

The target listens on `quic://127.0.0.1:5447/` by default, uses ALPN
`plab-raw-quic`, and honors `PROTOCOL_LAB_TARGET_BIND_ADDRESS`,
`PROTOCOL_LAB_TARGET_ADVERTISE_HOST`, and `PLAB_QUIC_PORT` for lab execution.

Local checks:

```powershell
go -C ./implementations/quic-go-raw/source test ./cmd/quic-go-raw
pwsh ./scripts/package/Build-QuicGoRawPackage.ps1
```
