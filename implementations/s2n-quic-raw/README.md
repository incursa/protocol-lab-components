# s2n-quic Raw QUIC

`s2n-quic-raw` packages a server built directly on AWS's independent s2n-quic
Rust transport implementation. It speaks ProtocolLab's raw QUIC ALPN and
stream wire contract and is deliberately separate from any HTTP/3 origin.

Package ID: `org.protocol-lab.components.implementation.s2n-quic-raw`

Pinned upstream and build inputs:

- s2n-quic `1.83.0` from upstream tag `v1.83.0`
- rustls TLS provider selected explicitly
- Cargo dependency graph in `source/Cargo.lock`
- Rust `1.88.0` Bookworm build image at
  `rust@sha256:af306cfa71d987911a781c37b59d7d67d934f49684058f96cf72079c3626bfe0`

Supported scenarios:

- `quic.transport.handshake-cold`
- `quic.transport.latency.echo-1kb`
- `quic.transport.stream-throughput.1mb`
- `quic.transport.multiplex.100x64kb`
- `quic.transport.duplex-streams`

The target listens on UDP port `5449` by default, negotiates ALPN
`plab-raw-quic`, and honors `PROTOCOL_LAB_TARGET_BIND_ADDRESS`,
`PROTOCOL_LAB_TARGET_ADVERTISE_HOST`, `PLAB_QUIC_PORT`, and
`PLAB_SCENARIO_ID`.

Build and validation:

```powershell
pwsh ./scripts/package/Build-S2nQuicRawPackage.ps1
pwsh ./scripts/package/Validate-ProtocolLabComponentManifests.ps1
```
