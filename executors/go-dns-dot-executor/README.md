# Go DNS-over-TLS executor

Package version `0.2.1` executes both strict `dns.dot.query.a` and authoritative `dns.dot.interoperability.query.a` with `secure-dns-smoke`. It establishes TLS 1.3 with ALPN `dot`, authenticates the packaged `dns.plab.test` leaf, sends the length-prefixed canonical A query, correlates the response, and validates canonical DNS semantics. The immutable patch version also emits and verifies the scenario-specific protocol variant used by the package-backed runner.

The executor performs no recursive lookup, fallback, cache use, or retry. TLS proof records Go platform provenance and an honest `not-reported` acceleration mode. Other DNS transports are explicitly unsupported, unknown identities fail closed, and local evidence is diagnostic and non-publishable.
