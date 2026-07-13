# Go DNS-over-QUIC local fixture authority

This implementation package exposes only `dns.doq.query.a`. It serves the
deterministic `dns.plab-test-a.canonical` answer over QUIC v1 with TLS 1.3,
`doq` ALPN, a two-octet network-order message length, and one transaction per
client-initiated bidirectional stream. It has no recursive resolver, cache,
external upstream, classic DNS, DoT, DoH, or generic raw-QUIC echo path.

The embedded certificate material is test-only. Local smoke evidence is
diagnostic and non-publishable.
