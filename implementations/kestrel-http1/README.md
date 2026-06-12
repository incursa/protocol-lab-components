# Kestrel HTTP/1 Implementation

Minimal Kestrel-backed HTTP/1.1 implementation package for the HTTP/1 core smoke slice.

Package identity is intentionally `kestrel-http1`. Future `kestrel-http2` and `kestrel-http3` packages should stay beside it instead of widening this package.

## Supported Slice

- Protocol: `h1` / HTTP/1.1 only
- Supported test case IDs and scenarios:
  - `http.core.plaintext`
  - `http.core.echo`
  - `http.core.json`
- Supported capabilities:
  - `http.server`
  - `httpPlaintext`
  - `httpEcho`
  - `httpJson`
- Known unsupported cases:
  - `h2`
  - `h3`
  - `https`
  - `websocket`
  - `server-sent-events`

## Endpoints

| Method | Path | Behavior |
| --- | --- | --- |
| `GET` | `/health` | Readiness probe. |
| `GET` | `/protocol-lab/metadata` | Startup and capability metadata. |
| `GET` | `/plaintext` | Returns `Hello, World!` as `text/plain`. |
| `GET` | `/json` | Returns `{"message":"Hello, World!"}` as `application/json`. |
| `POST` | `/echo` | Echoes the request body as `application/octet-stream`. |

## Local Run

```powershell
pwsh ./implementations/kestrel-http1/run.ps1 -Port 8080
```

```bash
PLAB_HTTP_PORT=8080 ./implementations/kestrel-http1/run.sh
```

Smoke the local server:

```powershell
Invoke-WebRequest http://127.0.0.1:8080/plaintext
Invoke-WebRequest http://127.0.0.1:8080/json
$body = [byte[]](0..255)
Invoke-WebRequest http://127.0.0.1:8080/echo -Method Post -ContentType application/octet-stream -Body $body
```

## Build Package

Build a self-contained package for the current Windows controller lane:

```powershell
pwsh ./scripts/package/Build-KestrelHttp1Package.ps1 -RuntimeIdentifier win-x64
```

Build a Linux package:

```powershell
pwsh ./scripts/package/Build-KestrelHttp1Package.ps1 -RuntimeIdentifier linux-x64
```

Artifacts are written under `artifacts/packages/` as `.plabpkg` files. The package entrypoint is the published `bin/kestrel-http1` process, so an adapter or controller can start it from package metadata without knowing it is implemented in .NET.
