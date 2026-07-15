# Go DNS-over-HTTPS HTTP/2 fixture authority

Package version `0.2.0` supports strict `dns.doh2.query.a` and authoritative `dns.doh2.interoperability.query.a`. It accepts the canonical query through exact HTTP/2 `POST /dns-query`, TLS 1.3, and ALPN `h2`, then returns the canonical authoritative response with `application/dns-message` and `Cache-Control: no-store`.

No recursive resolver, cache, external upstream, HTTP/1.1 route, or alternate DNS binding exists. The shared test certificate is implementation material and is never evidence.
