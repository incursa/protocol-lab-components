# Caddy HTTP/2 Implementation

This package provides an exact Caddy cleartext HTTP/2 prior-knowledge (`h2c`)
target for `http2.core.plaintext` and `http2.core.json`. It is compatible with
`go-http2-executor@0.3.0`, whose protocol proof rejects HTTP/1 fallback.

## Protocol boundary

- Supported: h2c prior knowledge on container port `8080`.
- Supported executor: `go-http2-executor`.
- Not supported by this package: TLS with ALPN `h2`, HTTP/2 upgrade from HTTP/1,
  HTTP/2 WebSocket, HTTP/1 package claims, or HTTP/3.

TLS/ALPN is intentionally not inferred from Caddy's general capabilities. It
needs a separate certificate-bearing target package and a TLS-capable executor.

## Pinned provenance and license

The Dockerfile pins `caddy:2.11.2-alpine` by OCI index digest
`sha256:834468128c7696cec0ceea6172f7d692daf645ae51983ca76e39da54a97c570d`.
For Linux x64 that index resolves to
`sha256:6d125e80883be8a2bef5f088c7535945b42cdbb8a0f5471bf36eaf18dc0638f1`.
Caddy tag `v2.11.2` resolves to source commit
`ffb6ab0644f24c5ee6542aca6bd59b7a1b0a8f91` and is Apache-2.0 licensed.
The exact upstream license text and machine-readable provenance ship in
`docker/third-party/` and are copied into the built image.

## Build and prove

```powershell
pwsh ./implementations/caddy-http2/run.ps1 -ProofOnly
pwsh ./scripts/package/Build-CaddyHttp2Package.ps1
```

## Live local executor smoke

```powershell
docker run --rm -d --name protocol-lab-caddy-http2 -p 8083:8080 incursa-protocol-lab-caddy-http2:0.1.0
pwsh ./executors/go-http2-executor/execute.ps1 `
  -TargetBaseUrl http://127.0.0.1:8083 `
  -OutputDirectory ./artifacts/caddy-http2-executor-smoke
docker rm -f protocol-lab-caddy-http2
```
