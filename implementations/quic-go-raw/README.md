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

- `quic.transport.stream-throughput.64kb`
- `quic.transport.stream-throughput.1mb`
- `quic.transport.stream-download.1mb`
- `quic.transport.stream-throughput.16mb`
- `quic.transport.sustained-stream.256x64kb`
- `quic.transport.sustained-stream.16384x1kb`
- `quic.transport.sustained-download.256x64kb`
- `quic.transport.sustained-download.16384x1kb`
- `quic.transport.sustained-download.4096x1kb`
- `quic.transport.latency.echo-1kb`
- `quic.transport.multiplex.100x1kb`
- `quic.transport.multiplex.100x64kb`
- `quic.transport.multiplex.16x1mb`
- `quic.transport.multiplex.mixed-4x16`
- `quic.transport.stream-limits.100x64kb`
- `quic.transport.flow-control.slow-reader-16x64kb`
- `quic.transport.connection-churn`
- `quic.transport.stream-churn`
- `quic.transport.duplex-streams`
- `quic.transport.duplex-streams.16x1mb`
- `quic.transport.duplex-streams-peer-matrix`
- `quic.transport.handshake-cold`

Unsupported until proven:

- HTTP/3 scenarios
- raw QUIC large-payload, cancellation, `quic.transport.resumption-rejected`, `quic.transport.resumed-handshake`, `quic.transport.zero-rtt-accepted`, and `quic.transport.zero-rtt-rejected` lanes

The package covers cold handshake, connection churn, stream churn, stream echo,
exact deterministic server-to-client download, and one-connection stream-limit
pressure lanes, including exact mixed-size echoes up to 1 MiB.

Version `0.1.19` adds exact 16MiB fixed-total download support using 16,384 sequential 1KiB target writes and advertises the paired upload lane.
Version `0.1.18` adds scenario-selected 1KiB target writes for the exact
4,096x1KiB sustained download while existing downloads retain 64KiB writes.

The target listens on `quic://127.0.0.1:5447/` by default, uses ALPN
`plab-raw-quic`, and honors `PROTOCOL_LAB_TARGET_BIND_ADDRESS`,
`PROTOCOL_LAB_TARGET_ADVERTISE_HOST`, and `PLAB_QUIC_PORT` for lab execution.

Local checks:

```powershell
go -C ./implementations/quic-go-raw/source test ./cmd/quic-go-raw
pwsh ./scripts/package/Build-QuicGoRawPackage.ps1
```
