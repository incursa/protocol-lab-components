# Go HTTP/1 Test Executor

`go-http1-executor` is a lane-scoped Protocol Lab test executor for cleartext HTTP/1.1 validation and load generation. It is not a fallback executor and it does not claim HTTP/2 or HTTP/3 coverage. Version `0.3.0` validates the deterministic workload first, then invokes the package's exact pinned `oha 1.15.0` PGO binary and emits normalized HTTP executor JSON.

## Supported

- Protocol family: `h1`
- Test cases: `http1.core.plaintext`, `http1.core.json`
- Scenarios: `http1.core.plaintext`, `http1.core.json`
- Validity: exact HTTP/1.1, status, media type, payload length, exact payload bytes, and SHA-256
- Load generator: embedded `oha 1.15.0` PGO binary for `win-x64` or `linux-x64`, verified by SHA-256 before execution
- Metrics: requests/second, application bytes/second, request outcomes, latency mean and p50/p75/p90/p95/p99, and status counts
- Artifacts: validation/protocol proof, executor and load-generator identities, normalized result, raw oha JSON/stderr, command, and version
- Raw stdout/stderr: preserved by the invoking runner or package host

## Known Unsupported

- HTTP/2
- HTTP/3
- HTTPS-required targets
- request-body echo
- payload-byte scenarios
- connection churn or disabled keep-alive profiles

The first performance slice supports duration-based clean-network load with HTTP/1.1 connection reuse. `bytesSent` is application request-payload bytes and is zero for the initial GET scenarios; it does not claim transport-header byte accounting. Package identity, tool version, tool SHA-256, raw stdout/stderr, and requested/effective load remain required evidence.

## Local Smoke

Start an HTTP/1 target on `127.0.0.1:8080`, then run:

```powershell
pwsh ./executors/go-http1-executor/execute.ps1 -TargetBaseUrl http://127.0.0.1:8080
```

Linux/macOS:

```bash
PLAB_TARGET_BASE_URL=http://127.0.0.1:8080 ./executors/go-http1-executor/execute.sh
```

Run the package-local tests:

```powershell
go -C ./executors/go-http1-executor/source test .
```

## Build Package

```powershell
pwsh ./scripts/package/Build-GoHttp1ExecutorPackage.ps1
pwsh ./scripts/package/Build-GoHttp1ExecutorPackage.ps1 linux-x64
```
