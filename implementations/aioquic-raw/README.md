# aioquic Raw QUIC

`aioquic-raw` packages a server built directly on aioquic's independent Python
QUIC implementation. It speaks ProtocolLab's raw QUIC ALPN and stream wire
contract and is deliberately separate from the aioquic HTTP/3 origin package.

Package ID: `org.protocol-lab.components.implementation.aioquic-raw`

Pinned upstream and build inputs:

- aioquic `1.3.0`
- PyInstaller `6.17.0` for the self-contained Linux x64 executable
- complete exact Python dependency versions in `source/requirements-build.txt`
- Python 3.12 Bookworm slim build image at
  `python:3.12-slim-bookworm@sha256:d50fb7611f86d04a3b0471b46d7557818d88983fc3136726336b2a4c657aa30b`

Supported scenarios:

- `quic.transport.handshake-cold`
- `quic.transport.latency.echo-1kb`
- `quic.transport.stream-throughput.1mb`
- `quic.transport.multiplex.100x64kb`
- `quic.transport.duplex-streams`

The target listens on UDP port `5451` by default, negotiates ALPN
`plab-raw-quic`, and honors `PROTOCOL_LAB_TARGET_BIND_ADDRESS`,
`PROTOCOL_LAB_TARGET_ADVERTISE_HOST`, `PLAB_QUIC_PORT`, and
`PLAB_SCENARIO_ID`. The adapter supplies only ProtocolLab's exact application
stream semantics; aioquic owns the QUIC and TLS protocol behavior.

Build and validation:

```powershell
pwsh ./scripts/package/Build-AioquicRawPackage.ps1
pwsh ./scripts/package/Validate-ProtocolLabComponentManifests.ps1
```
