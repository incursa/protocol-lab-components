# Go DNS-over-TLS executor

Package version `0.3.0` executes strict `dns.dot.query.a`, authoritative `dns.dot.interoperability.query.a`, and recursive-resolver `dns.dot.resolver.interoperability.query.a` with `secure-dns-smoke`. It establishes authenticated DoT with ALPN `dot`, sends the scenario-specific canonical A query, correlates the response, and validates exact DNS semantics. Resolver runs call the selected package's explicit local cache-control endpoint before every operation and emit resolver and local-upstream proof artifacts.

The executor performs no recursive lookup, fallback, cache use, or retry. TLS proof records Go platform provenance and an honest `not-reported` acceleration mode. Other DNS transports are explicitly unsupported, unknown identities fail closed, and local evidence is diagnostic and non-publishable.
