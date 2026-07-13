# Go DNS-over-HTTPS HTTP/2 executor

Package-backed test-side executor for exact `dns.doh2.query.a` / `secure-dns-smoke`. It establishes and validates TLS 1.3 with ALPN `h2`, then reuses the connection for exact HTTP/2 `POST /dns-query` operations with `application/dns-message` bodies.

The executor proves the canonical DNS semantics after parse and canonical reserialization, HTTP status and headers, no fallback, exact certificate identity, local-authoritative-only operation, and zero malformed, retry, failure, and timeout counts. Every other committed DNS scenario exits explicitly unsupported. Evidence is local diagnostic evidence and is not publishable or ranking-eligible.
