# nginx HTTP/2 Implementation

This package provides an exact nginx cleartext HTTP/2 prior-knowledge (`h2c`)
target for `http2.core.plaintext` and `http2.core.json`. The wrapper proves the
selected nginx build includes `--with-http_v2_module` before serving.

## Protocol boundary

- Supported: h2c prior knowledge on container port `8080`.
- Supported executor: `go-http2-executor@0.3.0`, which rejects HTTP/1 fallback.
- Not supported by this package: TLS with ALPN `h2`, HTTP/2 WebSocket, HTTP/1
  package claims, or HTTP/3.

nginx can parse HTTP/1 on a cleartext HTTP/2-enabled listener; that does not
make this an HTTP/1 package. ProtocolLab's executor must observe `HTTP/2.0` for
the declared scenarios to pass. TLS/ALPN needs a separate package and executor.

## Pinned provenance and license

The Dockerfile pins `nginx:1.29.0-alpine` by OCI index digest
`sha256:d67ea0d64d518b1bb04acde3b00f722ac3e9764b3209a9b0a98924ba35e4b779`.
For Linux x64 that index resolves to
`sha256:845b5424415de5f77dd5753cbb7c1be8bd8e44cc81f20f9705783a02f8848317`.
nginx tag `release-1.29.0` resolves to source commit
`235f409907fd60eb2d8f6ecdc0e5cb163dd6d45f` and is BSD-2-Clause licensed.
The exact upstream license text and machine-readable provenance ship in
`docker/third-party/` and are copied into the built image.

## Build and prove

```powershell
pwsh ./implementations/nginx-http2/run.ps1 -ProofOnly
pwsh ./scripts/package/Build-NginxHttp2Package.ps1
```

## Live local executor smoke

```powershell
docker run --rm -d --name protocol-lab-nginx-http2 -p 8084:8080 incursa-protocol-lab-nginx-http2:0.1.1
pwsh ./executors/go-http2-executor/execute.ps1 `
  -TargetBaseUrl http://127.0.0.1:8084 `
  -OutputDirectory ./artifacts/nginx-http2-executor-smoke
docker rm -f protocol-lab-nginx-http2
```
