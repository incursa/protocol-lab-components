# Go DoH3 local fixture authority

Package version `0.2.0` is a pinned `quic-go` HTTP/3 origin supporting the seven strict deterministic fixtures plus authoritative `dns.doh3.interoperability.query.a`. It permits exact POST and the canonical unpadded-base64url GET binding, disables cache, upstream, and recursion, and supports QUIC v1 with TLS 1.3 and `h3` ALPN only.
