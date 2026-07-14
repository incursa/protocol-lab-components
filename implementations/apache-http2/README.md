# Apache httpd HTTP/2 Implementation

This lane-scoped package runs the unmodified Apache HTTP Server and
`mod_http2` from a pinned Canonical image with static, hash-recorded
ProtocolLab fixtures.

## Exact supported slice

- Package: `org.protocol-lab.components.implementation.apache-http2@0.1.0`
- Implementation: `apache-http2`
- Scenarios: `http2.core.plaintext`, `http2.core.json`
- Default variant: `http2-h2c-prior-knowledge`
- Compatible executor: `go-http2-executor@0.3.0`

The package also exposes `http2-tls-alpn` with TLS 1.3 and negotiated ALPN
`h2` as a separate startup variant. That variant is implementation-supported
and locally protocol-smoked, but it is explicitly validation-unavailable and
non-ranking because `go-http2-executor@0.3.0` rejects HTTPS targets and no
compatible general HTTP/2 TLS executor package exists.

`/bytes/1024` and `/headers/response` are deterministic config/static-fixture
endpoints, but are not advertised as scenarios without exact executor rows.
CGI, custom modules, upload processing, server-side hashing, application
streaming, RFC 8441 origin behavior, H1 evidence, and H3 are unsupported.

## Runtime

The wrapper requires Docker with Linux-container support. It pulls only the
digest-pinned image in `toolchain.json`; no Apache or nghttp2 binary is bundled.
The committed private key is a public ProtocolLab test fixture for local
`apache.plab.test` TLS proof and is not a production credential.

Start h2c prior knowledge:

```powershell
pwsh ./implementations/apache-http2/run.ps1 -Variant h2c -Port 8082
```

Run the current exact executor in another shell:

```powershell
$env:PLAB_TARGET_BASE_URL = 'http://127.0.0.1:8082'
pwsh ./executors/go-http2-executor/execute.ps1 -OutputDirectory ./artifacts/apache-http2-h2c-source-smoke
```

Start the separate TLS/ALPN variant:

```powershell
pwsh ./implementations/apache-http2/run.ps1 -Variant tls-alpn -Port 8443
```

## Package build and extracted smoke

```powershell
pwsh ./scripts/package/Build-ApacheHttp2Package.ps1
pwsh ./scripts/package/Test-ApacheHttpImplementationPackages.ps1
```

The package-specific smoke materializes the scenario, executor, and target
archives, requires exact h2c executor validation, and separately proves TLS
1.3 plus ALPN `h2` through an exact-version .NET client. Raw executor outputs
are retained under `artifacts/apache-http-package-smoke`.
