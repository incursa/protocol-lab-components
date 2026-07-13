# DoH3 component delivery evidence

Authority: `protocol-lab@8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574`.

This component-only slice adds `org.protocol-lab.components.scenario.dns-doh3-performance@0.1.0`, `org.protocol-lab.components.executor.go-dns-doh3-executor@0.1.0`, and independent target `org.protocol-lab.components.implementation.go-dns-doh3@0.1.0`.

Exact supported identities are `dns.doh3.query.a`, `dns.doh3.get.a`, `dns.doh3.query.aaaa`, `dns.doh3.query.cname-chain`, `dns.doh3.query.large-dnssec-shaped`, `dns.doh3.query.nodata`, and `dns.doh3.query.nxdomain`. All other committed DNS transports return explicit `unsupported`; unknown IDs return configuration exit code `2`.

The package-local executor requires QUIC v1, TLS 1.3, ALPN `h3`, authenticated leaf DER/SPKI hashes, HTTP/3.0, exact authority/path/method/media/cache binding, no fallback, one reused connection, parsed semantic equivalence to the selected deterministic DNS fixture, canonical query and response hashes, and zero malformed/retried/failed/timed-out operations. The large case proves the exact 630-byte canonical response and explicitly records `dnssecSignatureValidity: not-claimed`.

The extracted Windows development smoke completed the following operations before the clean-source rebuild:

| Scenario | Completed | Malformed | Retries | Failed | Timed out | Canonical response bytes |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `dns.doh3.query.a` | 15,120 | 0 | 0 | 0 | 0 | 43 |
| `dns.doh3.get.a` | 16,623 | 0 | 0 | 0 | 0 | 43 |
| `dns.doh3.query.aaaa` | 15,918 | 0 | 0 | 0 | 0 | 55 |
| `dns.doh3.query.cname-chain` | 15,910 | 0 | 0 | 0 | 0 | 95 |
| `dns.doh3.query.large-dnssec-shaped` | 14,420 | 0 | 0 | 0 | 0 | 630 |
| `dns.doh3.query.nodata` | 15,380 | 0 | 0 | 0 | 0 | 104 |
| `dns.doh3.query.nxdomain` | 15,816 | 0 | 0 | 0 | 0 | 112 |

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
