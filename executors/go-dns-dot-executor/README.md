# Go DNS-over-TLS executor

Package-backed test-side executor for the exact `dns.dot.query.a` / `secure-dns-smoke` contract. It establishes TLS 1.3 with ALPN `dot`, authenticates the packaged `dns.plab.test` leaf by the package-local test root and exact certificate hashes, sends the length-prefixed `dns.plab-test-a.canonical` query with a nonzero runtime message ID, correlates the response, normalizes its ID to zero, and validates the canonical lengths, hashes, flags, answer, and zero TTL.

The executor performs no recursive lookup, fallback, cache use, or retry. It emits `validation.json`, `protocol-proof.json`, `dns-wire-summary.json`, `tls-negotiation.json`, `result.json`, `dns-dot-executor-result.json`, phase summaries, and exact executor/load-generator identities. The runner preserves stdout and stderr. Evidence is local diagnostic evidence and is not publishable or ranking-eligible.

All committed classic DNS, DoH2, DoH3, and DoQ scenario identities are explicitly unsupported. Unknown identities fail independently and cannot be substituted with `dns.dot.query.a`.
