# Kestrel HTTP/3 Implementation

`kestrel-http3` is a lane-scoped Protocol Lab implementation package for HTTP/3 over Kestrel. It is intentionally separate from `kestrel-http1` and `kestrel-http2` so inventory can select exactly the protocol lane under test.

## Supported

- Protocol family: `h3`
- Protocol version: `http/3`
- Test cases: `http3.core.status`, `http3.payload.bytes.64kb`, `http3.payload.bytes.1mb`
- Scenarios: `http3.core.status`, `http3.payload.bytes.64kb`, `http3.payload.bytes.1mb`

## Known Unsupported

- HTTP/1
- HTTP/2 and h2c
- raw QUIC
- WebSocket
- server-sent events

## Local Smoke

Install/trust a development certificate before the first run:

```powershell
dotnet dev-certs https --trust
```

Start the target:

```powershell
pwsh ./implementations/kestrel-http3/run.ps1 -Port 8443
```

Then use an HTTP/3-capable client against:

```text
https://127.0.0.1:8443/health
```

## Build Package

```powershell
pwsh ./scripts/package/Build-KestrelHttp3Package.ps1
```
