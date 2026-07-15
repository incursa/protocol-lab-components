# Go DNS-over-QUIC local fixture authority

Package version `0.2.0` supports strict `dns.doq.query.a` and authoritative `dns.doq.interoperability.query.a`. It serves the canonical local answer over QUIC v1 with TLS 1.3, `doq` ALPN, two-octet network-order message length, and one transaction per client-initiated bidirectional stream.

No recursive resolver, cache, external upstream, alternate DNS transport, or generic raw-QUIC echo path exists. Certificate material is test-only; local smoke evidence is diagnostic and non-publishable.
