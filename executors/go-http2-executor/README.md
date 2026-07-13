# Go HTTP/2 Test Executor

`go-http2-executor` is the package-owned HTTP/2 performance executor. Version `0.3.0` uses `golang.org/x/net/http2` directly to open cleartext HTTP/2 prior-knowledge connections. It has no HTTP/1 transport and cannot silently upgrade or fall back.

## Supported slice

- Protocol: exact HTTP/2 over cleartext h2c prior knowledge
- Scenarios: `http2.core.plaintext`, `http2.core.json`
- Load profiles: stable `http2-smoke`, `http2-diagnostic`, and `http2-comparison`
- Requested shapes: `1/1/1` for smoke, `1/8/8` for diagnostic, and `16/128/8` for comparison, expressed as connections/global concurrency/configured streams per connection
- Validity: exact protocol, endpoint, status, media type, payload length, exact bytes, and SHA-256
- Topology proof: actual dial count, maximum active operations, sampled active HTTP/2 streams, and peer-advertised stream limits
- Logical generator: `go-x-net-http2-h2c-load@0.3.0`, distinct from the executor identity
- Metrics: operation outcomes, requests/second, application bytes/second, latency mean and p50/p75/p90/p95/p99, mean time to first byte, and status counts

The package can execute the non-publishable `http2-comparison` load contract. A local or single-implementation run is still not a matched comparison, ranking, or publishable result. TLS/ALPN, HTTP/2 WebSocket, and streaming-response claims remain unsupported.

## Load-shape boundary

The internal package bridge passes profile-authored concurrency and request timeout independently from connections and configured streams. The executor fail-closes unless smoke is exactly `1/1/1`, 5 seconds measured, 1 second warmup, and a 5-second request timeout; diagnostic is exactly `1/8/8`, 10 seconds measured, 1 second warmup, and a 10-second request timeout; or comparison is exactly `16/128/8`, 30 seconds measured, 10 seconds warmup, a 10-second request timeout, and `balanced-round-robin` assignment. The comparison profile accepts repetition indices 1 through 3. Requested and effective topology, per-connection maxima, peer stream limits, and distribution identity are preserved in `http2-topology.json`.

## Local verification

```powershell
go -C ./executors/go-http2-executor/source test -count=1 .
pwsh ./executors/go-http2-executor/execute.ps1 -TargetBaseUrl http://127.0.0.1:8082
```

The wrapper runs validation only unless the `PLAB_DURATION_SECONDS` load context is present.

## Build package

```powershell
pwsh ./scripts/package/Build-GoHttp2ExecutorPackage.ps1
pwsh ./scripts/package/Build-GoHttp2ExecutorPackage.ps1 linux-x64
```

Dirty-source builds are diagnostic only and require `-AllowDirtySource`; they are not parity, publication, or ranking evidence.
