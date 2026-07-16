# quic-go HTTP/3 Implementation

`quic-go-http3` packages a quic-go HTTP/3 server target for ProtocolLab HTTP application comparisons. The package archive includes a generated Linux x64 process binary for live worker execution and keeps Docker wrapper scripts/image metadata for local containerized runs. It is separate from the `quic-go-raw-load` executor package so inventory can select HTTP/3 server behavior independently from raw QUIC load generation.

## Supported

- Protocol family: `h3`
- Role: server
- Public scenarios: `http3.core.status`, `http3.payload.bytes.1kb`, `http3.payload.bytes.64kb`, `http3.payload.bytes.1mb`, `http3.payload.stream.100x16kb`, `http3.headers.response-headers-50x32`, `http3.protocol.qpack-repeated-headers`
- Endpoints: `/health`, `/status`, `/protocol-lab/metadata`, `/plaintext`, `/json`, `/bytes/1024`, `/bytes/65536`, `/bytes/1048576`, `/stream/bytes?chunks=100&size=16384&delayMs=0`, `/headers/response?count=50&size=32`
- Capabilities: `http.server`, `httpStatus`, `httpBytes`, `httpStreaming`
- Not-found behavior: any unmapped path returns HTTP status `404`

## Known Unsupported

- HTTP/1 and HTTP/2 package behavior
- raw QUIC transport scenarios; use `org.protocol-lab.components.executor.quic-go-raw-load` for raw QUIC load generation
- deterministic header-heavy/QPACK fixtures
- WebSocket-over-H3 and WebTransport
- h3spec/QPACK conformance proof for this target

## Pinned Toolchain

- Go toolchain expectation: `1.25.x`
- quic-go module: `github.com/quic-go/quic-go v0.60.0`
- quic-go source tag commit: `7612ad1036957cde45d672bebb8b9ec9c2e9b2a7`
- Builder image: `golang:1.25-bookworm`
- Builder image digest: `golang@sha256:bbb255b0e131db500cf0520adc97441d2260cf629c7fa7e39e025ddf53995a24`
- Runtime image: `scratch`
- Package version: `0.1.7`
- Component image tag: `incursa-protocol-lab-quic-go-http3:0.1.7`
- Host requirements: Docker and an HTTP/3-capable lab worker. The Go 1.25.x
  toolchain and quic-go v0.60.0 identity are pinned and retained inside the
  target image provenance; they are not host capability requirements.
- Component image ID: `sha256:68ad1269e2b02439bf796f95fe4d0009a1d7eb4e7dbbd0d173cdc83f58843edd`
- Component repo digest: `incursa-protocol-lab-quic-go-http3@sha256:68ad1269e2b02439bf796f95fe4d0009a1d7eb4e7dbbd0d173cdc83f58843edd`
- Certificate mode: generated self-signed loopback certificate

## Local Smoke

Plan the wrapper command without Docker execution:

```powershell
pwsh ./implementations/quic-go-http3/run.ps1 -PlanOnly
```

Build the wrapper image:

```powershell
docker build --pull `
  --build-arg QUIC_GO_VERSION=v0.60.0 `
  -f ./implementations/quic-go-http3/docker/quic-go-http3.Dockerfile `
  -t incursa-protocol-lab-quic-go-http3:0.1.7 `
  ./implementations/quic-go-http3
```

Start the server:

```powershell
pwsh ./implementations/quic-go-http3/run.ps1 -SkipBuild -Port 5446
```

Use an HTTP/3-capable client against:

```text
https://localhost:5446/health
https://localhost:5446/bytes/1024
https://localhost:5446/bytes/65536
https://localhost:5446/bytes/1048576
https://localhost:5446/stream/bytes?chunks=100&size=16384&delayMs=0
```

Linux/macOS plan-only smoke:

```bash
PLAB_PLAN_ONLY=1 ./implementations/quic-go-http3/run.sh
```

## Build Package

```powershell
pwsh ./scripts/package/Build-QuicGoHttp3Package.ps1
```
