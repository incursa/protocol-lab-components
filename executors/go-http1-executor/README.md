# Go HTTP/1 Test Executor

`go-http1-executor` is a lane-scoped Protocol Lab test executor for HTTP/1 target validation. It is not a fallback executor and it does not claim HTTP/2 or HTTP/3 coverage.

## Supported

- Protocol family: `h1`
- Test cases: `http.core.plaintext`, `http.core.json`
- Scenarios: `http.core.plaintext`, `http.core.json`
- Artifacts: `validation.json`, `result.json`, `load-tool-execution.json`

## Known Unsupported

- HTTP/2
- HTTP/3
- HTTPS-required targets
- request-body echo
- payload-byte scenarios
- load generation

## Local Smoke

Start an HTTP/1 target on `127.0.0.1:8080`, then run:

```powershell
pwsh ./executors/go-http1-executor/execute.ps1 -TargetBaseUrl http://127.0.0.1:8080
```

Linux/macOS:

```bash
PLAB_TARGET_BASE_URL=http://127.0.0.1:8080 ./executors/go-http1-executor/execute.sh
```

## Build Package

```powershell
pwsh ./scripts/package/Build-GoHttp1ExecutorPackage.ps1
```
