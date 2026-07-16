# quiche Raw QUIC

`quiche-raw` packages a server built directly on Cloudflare's quiche transport
library. It speaks ProtocolLab's raw QUIC ALPN and stream wire contract and is
deliberately separate from the Docker-backed quiche HTTP/3 peer package.

Package ID: `org.protocol-lab.components.implementation.quiche-raw`

Pinned inputs:

- quiche `0.29.3`
- exact Rust dependencies in `source/Cargo.lock`
- Rust `1.88.0` Bookworm image at
  `rust@sha256:af306cfa71d987911a781c37b59d7d67d934f49684058f96cf72079c3626bfe0`
- Debian Bookworm slim runtime at
  `debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818`

Supported scenarios:

- `quic.transport.handshake-cold`
- `quic.transport.latency.echo-1kb`
- `quic.transport.stream-throughput.1mb`
- `quic.transport.multiplex.100x64kb`
- `quic.transport.duplex-streams`

The target listens on UDP port `5452` by default, negotiates ALPN
`plab-raw-quic`, and honors `PROTOCOL_LAB_TARGET_BIND_ADDRESS`,
`PLAB_QUIC_PORT`, `PLAB_CERT_FILE`, `PLAB_KEY_FILE`, and `PLAB_SCENARIO_ID`.
The adapter supplies only ProtocolLab's exact application stream semantics;
quiche owns QUIC packet processing, recovery, congestion control, and TLS.

Build and validation:

```powershell
pwsh ./scripts/package/Build-QuicheRawPackage.ps1
pwsh ./scripts/package/Validate-ProtocolLabComponentManifests.ps1
```
