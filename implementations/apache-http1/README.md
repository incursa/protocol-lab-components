# Apache httpd HTTP/1.1 Implementation

This lane-scoped package runs the unmodified Apache HTTP Server from a pinned
Canonical image with static, hash-recorded ProtocolLab fixtures. It is a
general-purpose HTTP origin row, not a CGI application or custom module.

## Exact supported slice

- Package: `org.protocol-lab.components.implementation.apache-http1@0.1.0`
- Implementation: `apache-http1`
- Protocol: exact HTTP/1.1
- Executor: `go-http1-executor@0.3.0`
- Scenarios: `http1.core.plaintext`, `http1.core.json`

`/bytes/1024` and `/headers/response` are deterministic config/static-fixture
endpoints, but are not advertised as supported scenarios because the current
HTTP/1 executor does not execute those contract rows. Upload processing,
server-side hashing, echo, application streaming, WebSocket, TLS, H2, and H3
are explicitly unsupported.

## Runtime

The wrapper requires Docker with Linux-container support. It pulls only the
digest-pinned image in `toolchain.json`; no Apache binary is bundled.

```powershell
pwsh ./implementations/apache-http1/run.ps1 -Port 8080
```

In another shell:

```powershell
$env:PLAB_TARGET_BASE_URL = 'http://127.0.0.1:8080'
pwsh ./executors/go-http1-executor/execute.ps1 -OutputDirectory ./artifacts/apache-http1-source-smoke
```

## Package build and extracted smoke

```powershell
pwsh ./scripts/package/Build-ApacheHttp1Package.ps1
pwsh ./scripts/package/Test-ApacheHttpImplementationPackages.ps1
```

The package-specific smoke materializes the scenario, executor, and target
archives, starts this extracted wrapper, requires exact HTTP/1.1 validation,
and retains executor raw artifacts under `artifacts/apache-http-package-smoke`.
