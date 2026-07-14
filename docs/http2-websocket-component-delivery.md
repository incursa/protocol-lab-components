# HTTP/2 RFC 8441 WebSocket component delivery

Authority: `protocol-lab@8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574`.

This components-only correction retains the authority-locked scenario package
`org.protocol-lab.components.scenario.http2-websocket-performance@0.1.1`, updates
`org.protocol-lab.components.executor.go-http2-websocket-executor` and its
`go-x-net-http2-websocket-load` generator identity to `0.2.0`, and retains the
independent `org.protocol-lab.components.implementation.kestrel-http2-websocket@0.1.0`
target. The target version did not change because its existing RFC 8441 endpoint
already accepts multiple concurrent extended-CONNECT streams and preserves the
required frame semantics.

The exact supported identities are:

- `http2.websocket.rfc8441.extended-connect`
- `http2.websocket.rfc8441.control-frames`
- `http2.websocket.rfc8441.text-echo`
- `http2.websocket.rfc8441.binary-echo`
- `http2.websocket.rfc8441.close`
- `http2.websocket.rfc8441.multi-message-text-echo`

The first five identities retain the exact `websocket-smoke` profile. The
multi-message identity uses the exact authority `diagnostic` profile: one
authenticated TLS 1.3 connection with ALPN `h2`, eight concurrent RFC 8441
streams, concurrency eight, ten measured seconds, one-second warmup and
cooldown, one repetition, and a ten-second operation timeout. Every measured
operation on every stream carries exactly 100 ordered `protocol-lab` text
messages. Configured connection, stream, and concurrency capacity is reported
separately from observed active connections, streams, and effective
concurrency.

The executor proves TLS and certificate identity, ALPN `h2`, HTTP/2 extended
CONNECT pseudo-headers, `SETTINGS_ENABLE_CONNECT_PROTOCOL=1`, response status
200, prohibited-header absence, client masking, unmasked server frames, exact
payload bytes and hash, strict per-stream ordering, and clean close behavior.
It preserves raw stdout/stderr and writes the executor identity, generator
identity, protocol proof, validation, load summary, warmup summary, frame
summary, topology, and normalized result artifacts. HTTP/1.1 and HTTP/3
WebSocket identities fail closed as explicit `unsupported`; unknown identities
remain distinct configuration errors with exit code 2.

## Clean package evidence

Clean packages were built from implementation commit
`77d7182b00796151c3bf94168d9389ba38599809`. Every matching build attestation
records a clean source tree, is parity-eligible, and passes
`Test-ProtocolLabPackageBuildAttestation.ps1 -RequireParityEligible`. Source and
extracted package-v2 manifests agree on package ID, version, kind, entry
manifests, and the single runtime environment where applicable.

| Package artifact | SHA-256 |
| --- | --- |
| Scenario, portable | `f2be46e79b7afcfe9b81a9db5330d2af2fb61a59c02adb08ddaf0e0c44f378d7` |
| Executor, Windows x64 | `bc12595d4da23e50fa1fad38a3c818b8e5316963ceea3f1b36573e7d84f61245` |
| Executor, Linux x64 | `3c5dc8487fbd5a9f0b5624c2806be8c942e4d0fd91e87bad0d123132da2f7579` |
| Target, Windows x64 | `b7567ea2e064f27333a9fe8e0c02307380bed95805c941b71135a7c2c61eda3c` |
| Target, Linux x64 | `2baea874b3c1f805210234b206f89b78cc8f8d6dc497688b4cbccdba0b3e09da` |

The clean archives are rooted at `artifacts/http2-websocket-packages`. Extracted
Windows and Linux six-cell smokes are rooted at
`artifacts/http2-websocket-extracted-smoke-win-x64-final` and
`artifacts/http2-websocket-extracted-smoke-linux-x64-final`.

| Runtime | Scenario | Completed operations | Completed messages | Failed | Timed out | Observed connections / streams / concurrency |
| --- | --- | ---: | ---: | ---: | ---: | --- |
| Windows x64 | `http2.websocket.rfc8441.extended-connect` | 1,250 | 1,250 | 0 | 0 | 1 / 1 / 1 |
| Windows x64 | `http2.websocket.rfc8441.control-frames` | 1,244 | 1,244 | 0 | 0 | 1 / 1 / 1 |
| Windows x64 | `http2.websocket.rfc8441.text-echo` | 1,230 | 1,230 | 0 | 0 | 1 / 1 / 1 |
| Windows x64 | `http2.websocket.rfc8441.binary-echo` | 1,220 | 1,220 | 0 | 0 | 1 / 1 / 1 |
| Windows x64 | `http2.websocket.rfc8441.close` | 1,340 | 1,340 | 0 | 0 | 1 / 1 / 1 |
| Windows x64 | `http2.websocket.rfc8441.multi-message-text-echo` | 1,089 | 108,900 | 0 | 0 | 1 / 8 / 8 |
| Linux x64 | `http2.websocket.rfc8441.extended-connect` | 1,258 | 1,258 | 0 | 0 | 1 / 1 / 1 |
| Linux x64 | `http2.websocket.rfc8441.control-frames` | 1,598 | 1,598 | 0 | 0 | 1 / 1 / 1 |
| Linux x64 | `http2.websocket.rfc8441.text-echo` | 1,588 | 1,588 | 0 | 0 | 1 / 1 / 1 |
| Linux x64 | `http2.websocket.rfc8441.binary-echo` | 1,577 | 1,577 | 0 | 0 | 1 / 1 / 1 |
| Linux x64 | `http2.websocket.rfc8441.close` | 1,941 | 1,941 | 0 | 0 | 1 / 1 / 1 |
| Linux x64 | `http2.websocket.rfc8441.multi-message-text-echo` | 1,012 | 101,200 | 0 | 0 | 1 / 8 / 8 |

Each runtime also produced explicit unsupported evidence for all 18 enumerated
HTTP/1.1 and HTTP/3 WebSocket identities outside this executor lane. Unknown
`http2.websocket.unknown` input returned exit code 2 and did not create a
supported or unsupported result. Target logs show matching accepted and cleanly
closed stream counts: 7,600 on Windows and 9,443 on Linux. No exact Windows or
WSL package process remained after either smoke.

Verification commands:

```powershell
go -C executors/go-http2-websocket-executor/source test -race -count=1 ./...
go -C executors/go-http2-websocket-executor/source vet ./...
dotnet build implementations/kestrel-http2-websocket/source/KestrelHttp2WebSocket.csproj -c Release -p:TreatWarningsAsErrors=true
pwsh ./scenarios/http2-websocket-performance/validate.ps1
pwsh ./scripts/package/Validate-ProtocolLabComponentManifests.ps1
pwsh ./scripts/package/Test-ProtocolLabPackageBuildAttestation.ps1 -PackagePath <package> -AttestationPath <attestation> -RequireParityEligible
pwsh ./scripts/package/Test-Http2WebSocketThreePackageSmoke.ps1 -RuntimeIdentifier win-x64 -PackageRoot ./artifacts/http2-websocket-packages -ArtifactRoot ./artifacts/http2-websocket-extracted-smoke-win-x64-final
pwsh ./scripts/package/Test-Http2WebSocketThreePackageSmoke.ps1 -RuntimeIdentifier linux-x64 -PackageRoot ./artifacts/http2-websocket-packages -ArtifactRoot ./artifacts/http2-websocket-extracted-smoke-linux-x64-final
```

This delivery is local diagnostic component evidence only. It makes no
runner-admission, benchmark, comparison, ranking, publication, deployment, or
lab-infrastructure claim.
