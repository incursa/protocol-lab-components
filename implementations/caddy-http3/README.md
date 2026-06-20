# Caddy HTTP/3 Implementation

`caddy-http3` packages a Docker-backed Caddy HTTP/3 server target for ProtocolLab HTTP application comparisons. It is separate from `caddy-http1` so controller inventory can select the exact protocol lane.

## Supported

- Protocol family: `h3`
- Role: server
- Public scenarios: `http3.core.status`, `http3.payload.bytes.1kb`, `http3.payload.bytes.64kb`
- Endpoints: `/status`, `/plaintext`, `/json`, `/protocol-lab/metadata`, `/bytes/1024`, `/bytes/65536`

## Known Unsupported

- raw QUIC transport
- HTTP/1 and HTTP/2 package behavior
- HTTP/3 large body until package-backed validation covers the row
- deterministic header-heavy/QPACK fixtures
- WebSocket-over-H3
- h3spec/QPACK conformance proof for this target

## Pinned Toolchain

- Base image: `caddy:2.11.2-alpine`
- Linux amd64 image digest: `caddy@sha256:6d125e80883be8a2bef5f088c7535945b42cdbb8a0f5471bf36eaf18dc0638f1`
- Component image tag: `incursa-protocol-lab-caddy-http3:0.1.2`
- Certificate mode: Caddy `tls internal`

## Local Smoke

Plan the wrapper command without Docker execution:

```powershell
pwsh ./implementations/caddy-http3/run.ps1 -PlanOnly
```

Build the wrapper image:

```powershell
docker build --pull `
  -f ./implementations/caddy-http3/docker/Caddy.Dockerfile `
  -t incursa-protocol-lab-caddy-http3:0.1.2 `
  ./implementations/caddy-http3/docker
```

Start the server:

```powershell
pwsh ./implementations/caddy-http3/run.ps1 -SkipBuild -Port 5445
```

Use an HTTP/3-capable client against:

```text
https://localhost:5445/plaintext
```

Linux/macOS plan-only smoke:

```bash
PLAB_PLAN_ONLY=1 ./implementations/caddy-http3/run.sh
```

## Build Package

```powershell
pwsh ./scripts/package/Build-CaddyHttp3Package.ps1
```
