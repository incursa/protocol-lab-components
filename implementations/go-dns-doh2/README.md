# Go DNS-over-HTTPS HTTP/2 fixture authority

This independent target implements only `dns.doh2.query.a`. It accepts the canonical 27-byte zero-ID DNS query through exact `POST /dns-query` over HTTP/2, TLS 1.3, and ALPN `h2`, then returns the canonical 43-byte authoritative response with `application/dns-message` and `Cache-Control: no-store`.

It packages no recursive resolver, cache, external upstream, HTTP/1.1 route, or alternate DNS binding. The shared ProtocolLab test certificate is implementation material and is never benchmark evidence.
