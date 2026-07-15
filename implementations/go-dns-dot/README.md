# Go DNS-over-TLS local fixture authority

Package version `0.2.0` supports strict `dns.dot.query.a` and authoritative `dns.dot.interoperability.query.a` with the same local canonical fixture. It serves two-octet length-prefixed DNS over TLS 1.3 with ALPN `dot`; a valid `plab.test. IN A` request receives authoritative `192.0.2.1` with TTL zero.

No recursive resolver, external upstream, cache, alternate DNS transport, or fallback exists. The packaged test certificate and private key are implementation material and must never be copied into evidence.
