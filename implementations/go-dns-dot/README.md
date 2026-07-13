# Go DNS-over-TLS local fixture authority

This target is deliberately narrow. It serves only the exact `dns.plab-test-a.canonical` request over two-octet length-prefixed DNS on TLS 1.3 with ALPN `dot`. A valid request for `plab.test. IN A` receives the authoritative `192.0.2.1` answer with TTL zero. The runtime message ID is copied into the response and cannot be reused on a connection.

No recursive resolver, external upstream, cache, UDP socket, classic DNS/TCP listener, HTTP endpoint, QUIC endpoint, or fallback code exists. Every other committed DNS scenario is explicitly unsupported.

The packaged ECDSA P-256 key and certificate are test-only implementation material for `dns.plab.test`. The private key must never be copied into evidence.
