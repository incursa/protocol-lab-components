# picoquic Raw QUIC

`picoquic-raw` packages a native C server built directly on the independent
picoquic QUIC transport implementation. It speaks ProtocolLab's raw QUIC ALPN
and stream wire contract and is deliberately separate from any HTTP/3 origin.

Package ID: `org.protocol-lab.components.implementation.picoquic-raw`

Pinned upstream and build inputs:

- picoquic commit `13671ce7bdf58c278a29da2d49a32f76c21d6c6d`
- picotls commit `bfa67875982afc4c24f21e146cef4747fa189c2f`
- Debian 12 build image at
  `debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818`
- apt package versions pinned in `source/Dockerfile`

Supported scenarios:

- `quic.transport.handshake-cold`
- `quic.transport.latency.echo-1kb`
- `quic.transport.stream-throughput.1mb`
- `quic.transport.multiplex.100x64kb`
- `quic.transport.duplex-streams`

The target listens on UDP port `5450` by default, negotiates ALPN
`plab-raw-quic`, and honors `PROTOCOL_LAB_TARGET_ADVERTISE_HOST`,
`PLAB_QUIC_PORT`, `PLAB_SCENARIO_ID`, `PLAB_CERT_FILE`, and `PLAB_KEY_FILE`.
The adapter only supplies ProtocolLab's byte-for-byte application stream
semantics; picoquic and its pinned picotls/OpenSSL provider own the transport
and TLS protocol behavior.

Build and validation:

```powershell
pwsh ./scripts/package/Build-PicoquicRawPackage.ps1
pwsh ./scripts/package/Validate-ProtocolLabComponentManifests.ps1
```
