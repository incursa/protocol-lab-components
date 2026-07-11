# quic-go Raw QUIC

`quic-go-raw` packages a quic-go raw QUIC transport target for ProtocolLab raw
QUIC comparisons and handshake/churn validation. It is intentionally separate
from the quic-go HTTP/3 implementation and from the `quic-go-raw-load`
executor package.

Package ID: `org.protocol-lab.components.implementation.quic-go-raw`

The package archive carries both `bin/linux-x64/quic-go-raw` and
`bin/windows-x64/quic-go-raw.exe`. `run.sh` selects the Linux binary, `run.ps1`
selects the Windows binary, and the public YAML keeps the Linux executable as
the canonical implementation entrypoint.

Supported scenarios:

- `quic.transport.stream-throughput.1mb`
- `quic.transport.multiplex.100x64kb`
- `quic.transport.stream-churn`
- `quic.transport.duplex-streams`
- `quic.transport.cold-handshake`

Unsupported until proven:

- HTTP/3 scenarios
- raw QUIC latency, stream-limit, large-payload, cancellation, `quic.transport.resumption-rejected`, `quic.transport.resumed-handshake`, `quic.transport.zero-rtt-accepted`, and `quic.transport.zero-rtt-rejected` lanes

The package covers cold handshake, stream churn, and stream echo lanes.

The target listens on `quic://127.0.0.1:5447/` by default, uses ALPN
`plab-raw-quic`, and honors `PROTOCOL_LAB_TARGET_BIND_ADDRESS`,
`PROTOCOL_LAB_TARGET_ADVERTISE_HOST`, and `PLAB_QUIC_PORT` for lab execution.

Local checks:

```powershell
go -C ./implementations/quic-go-raw/source test ./cmd/quic-go-raw
pwsh ./scripts/package/Build-QuicGoRawPackage.ps1
```
