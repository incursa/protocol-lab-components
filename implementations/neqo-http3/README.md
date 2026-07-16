# Mozilla neqo HTTP/3 peer

`neqo-http3` packages the digest-pinned upstream neqo QNS image as a
server-only ProtocolLab target for `http3.external.peer-characterization`.
The target imports a generated certificate into an ephemeral NSS database and
starts `neqo-server` on UDP 4433. A request to `/` returns HTTP/3 status 200.

This is not a raw QUIC package. `neqo-server --help` states that the test
server remains HTTP/3 regardless of an overridden ALPN. The exact raw decision
is retained in `docs/raw-quic-interop-image-feasibility-2026-07-16.md`.

The package references Mozilla's immutable upstream image and does not
redistribute its binaries. It declares no canonical payload, JSON, QPACK,
WebSocket, or ranking row.

```powershell
pwsh ./scripts/package/Build-NeqoHttp3Package.ps1
pwsh ./implementations/neqo-http3/run.ps1 -PlanOnly
```
