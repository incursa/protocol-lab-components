# Kestrel HTTP/2 Implementation

Lane-scoped Kestrel HTTP/2 implementation package. This package is not a
combined Kestrel package; select `kestrel-http2` only for HTTP/2 work.

## Supported Slice

- Protocol: `h2`
- Transport: cleartext HTTP/2 prior-knowledge (`h2c`) on the configured port
- Scenarios:
  - `http2.core.plaintext`
  - `http2.core.json`
- Known unsupported:
  - `h1`
  - `h3`
  - TLS-required HTTP/2 clients
  - websocket
  - server-sent-events

## Local Smoke

Start locally:

```powershell
pwsh ./implementations/kestrel-http2/run.ps1 -Port 8082
```

Smoke with a client that can force HTTP/2 prior knowledge:

```powershell
dotnet run --project ./implementations/kestrel-http2/src/KestrelHttp2.csproj --configuration Release --no-launch-profile
```

In another shell, use a ProtocolLab HTTP/2-capable executor or curl build with
HTTP/2 prior-knowledge support against `http://127.0.0.1:8082/plaintext`.

## Build Package

```powershell
pwsh ./scripts/package/Build-KestrelHttp2Package.ps1
```
