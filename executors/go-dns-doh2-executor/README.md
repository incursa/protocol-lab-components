# Go DNS-over-HTTPS HTTP/2 executor

Package version `0.3.0` executes strict `dns.doh2.query.a`, authoritative `dns.doh2.interoperability.query.a`, and recursive-resolver `dns.doh2.resolver.interoperability.query.a` with `secure-dns-smoke`. It establishes authenticated TLS with ALPN `h2`, reuses the connection for exact HTTP/2 `POST /dns-query`, rejects HTTP/1 fallback, and binds the reported protocol, TLS, certificate, DNS role, cache-control, and local-upstream proofs to the selected scenario.

Strict v1 continues to require response `Cache-Control: no-store`; interoperability accepts its absence as permitted by RFC 8484. Both lanes require the DNS media type and canonical authoritative DNS semantics. TLS proof records Go platform provenance and an honest `not-reported` acceleration mode. Other DNS transports fail closed; local evidence is diagnostic and non-publishable.
