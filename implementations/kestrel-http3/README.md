# Kestrel HTTP/3 Implementation

`kestrel-http3` is a lane-scoped Protocol Lab implementation package for HTTP/3 over Kestrel. It is intentionally separate from `kestrel-http1` and `kestrel-http2` so inventory can select exactly the protocol lane under test.

- Package version: `0.1.4`

## Supported

- Protocol family: `h3`
- Protocol version: `http/3`
- Test cases: `http3.core.status`, `http3.payload.bytes.1kb`, `http3.payload.bytes.64kb`, `http3.payload.bytes.1mb`
- Scenarios: `http3.core.status`, `http3.payload.bytes.1kb`, `http3.payload.bytes.64kb`, `http3.payload.bytes.1mb`

## Known Unsupported

- HTTP/1
- HTTP/2 and h2c
- raw QUIC
- WebSocket
- server-sent events

## Local Smoke

The target creates a short-lived, loopback-only self-signed certificate for
each process. Start it through the same cross-platform entrypoint used by the
package:

```powershell
dotnet run --project ./implementations/kestrel-http3/src/KestrelHttp3.csproj --no-launch-profile
```

Then use an HTTP/3-capable client against:

```text
https://127.0.0.1:8443/health
```

## Build Package

```powershell
pwsh ./scripts/package/Build-KestrelHttp3Package.ps1
```
