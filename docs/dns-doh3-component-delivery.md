# DoH3 component delivery evidence

Authority: `protocol-lab@8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574`.

This component-only slice adds `org.protocol-lab.components.scenario.dns-doh3-performance@0.1.0`, `org.protocol-lab.components.executor.go-dns-doh3-executor@0.1.0`, and independent target `org.protocol-lab.components.implementation.go-dns-doh3@0.1.0`.

Exact supported identities are `dns.doh3.query.a`, `dns.doh3.get.a`, `dns.doh3.query.aaaa`, `dns.doh3.query.cname-chain`, `dns.doh3.query.large-dnssec-shaped`, `dns.doh3.query.nodata`, and `dns.doh3.query.nxdomain`. All other committed DNS transports return explicit `unsupported`; unknown IDs return configuration exit code `2`.

The package-local executor requires QUIC v1, TLS 1.3, ALPN `h3`, authenticated leaf DER/SPKI hashes, HTTP/3.0, exact authority/path/method/media/cache binding, no fallback, one reused connection, parsed semantic equivalence to the selected deterministic DNS fixture, canonical query and response hashes, and zero malformed/retried/failed/timed-out operations. The large case proves the exact 630-byte canonical response and explicitly records `dnssecSignatureValidity: not-claimed`.

Clean packages were built from component commit `0500afb4b95f465fbed0d475ea7b92b6cbce728d`. All matching build attestations report clean source, are parity-eligible, and pass `Test-ProtocolLabPackageBuildAttestation.ps1 -RequireParityEligible`.

| Package artifact | SHA-256 |
| --- | --- |
| Scenario, portable | `c905f4fab992f6379547152bf2e4516ba12439da50375b54585ffdb1776a6068` |
| Executor, Windows x64 | `4dcd5dfa9524297574acd319ad6efeafb4f83b73093dc0736ed2f127ee9ed1a1` |
| Executor, Linux x64 | `61030aba86add8bb414c01e701b5b59074b7985863c93702c43d2a5520be6e20` |
| Target, Windows x64 | `f8687e9e048b96f4937c9665fa94836b1f823251029fb68a0ddc3c177b655069` |
| Target, Linux x64 | `b82bc7654c4983649be050ab93b6074336129603aae6a3ef629947c5f2cb18dd` |

The final extracted Windows smoke from those clean archives is rooted at `artifacts/dns-doh3-three-package-smoke-clean-0500afb` and completed:

| Scenario | Completed | Malformed | Retries | Failed | Timed out | Canonical response bytes |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `dns.doh3.query.a` | 15,535 | 0 | 0 | 0 | 0 | 43 |
| `dns.doh3.get.a` | 13,399 | 0 | 0 | 0 | 0 | 43 |
| `dns.doh3.query.aaaa` | 14,238 | 0 | 0 | 0 | 0 | 55 |
| `dns.doh3.query.cname-chain` | 15,380 | 0 | 0 | 0 | 0 | 95 |
| `dns.doh3.query.large-dnssec-shaped` | 14,752 | 0 | 0 | 0 | 0 | 630 |
| `dns.doh3.query.nodata` | 16,548 | 0 | 0 | 0 | 0 | 104 |
| `dns.doh3.query.nxdomain` | 15,810 | 0 | 0 | 0 | 0 | 112 |

Verification commands:

```powershell
go -C implementations/go-dns-doh3/source test -race -count=1 ./...
go -C implementations/go-dns-doh3/source vet ./...
go -C executors/go-dns-doh3-executor/source test -race -count=1 ./...
go -C executors/go-dns-doh3-executor/source vet ./...
pwsh ./scripts/package/Validate-ProtocolLabComponentManifests.ps1
pwsh ./scripts/package/Test-DnsDoh3ThreePackageSmoke.ps1
```

The `dns-doh3-performance-comparison` suite and `secure-dns-comparison` profile are packaged only as authority-locked, non-publishable contract material. The executor deliberately rejects that profile until a controlled comparison slice implements its `4/32`, 30-second, five-repetition topology and saturation evidence. Nothing here is a benchmark, comparison, ranking, runner-admission, publication, or deployment claim.
