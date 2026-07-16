# XQUIC HTTP/3 peer

`xquic-http3` packages the digest-pinned upstream XQUIC interop image as a
server-only target for `http3.external.peer-characterization`. It is paired
with the separately packaged `xquic-http3-client` executor.

The XQUIC pair negotiates HTTP/3 and observes status 200. The current upstream
sample does not deliver a response FIN in the tested root transaction; that
warning is retained in the client log and prevents canonical payload or
ranking claims. The generic curl peer also reports a completion error after
the status response.

This is not a raw QUIC package. The exact decision is retained in
`docs/raw-quic-interop-image-feasibility-2026-07-16.md`.

```powershell
pwsh ./scripts/package/Build-XquicHttp3Package.ps1
pwsh ./implementations/xquic-http3/run.ps1 -PlanOnly
```
