# LSQUIC HTTP/3 peer

`lsquic-http3` packages the digest-pinned upstream LSQUIC interop image as a
server-only ProtocolLab target for `http3.external.peer-characterization`.
The package starts `http_server` in HTTP/3 mode and serves a diagnostic root
with status 200.

This is not a raw QUIC package. The upstream `-Q` option selects HQ
file-server semantics even when a custom ALPN is supplied. The exact raw
decision is retained in
`docs/raw-quic-interop-image-feasibility-2026-07-16.md`.

The package declares no canonical payload, JSON, QPACK, WebSocket, or ranking
row. LSQUIC's MIT license and bundled Chromium notices remain owned by the
upstream image; this package references the immutable image and redistributes
neither its binaries nor its filesystem layers.

Build and plan a local launch:

```powershell
pwsh ./scripts/package/Build-LsquicHttp3Package.ps1
pwsh ./implementations/lsquic-http3/run.ps1 -PlanOnly
```
