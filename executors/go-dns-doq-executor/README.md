# Go DNS-over-QUIC test executor

Package version `0.2.0` executes strict `dns.doq.query.a` and authoritative `dns.doq.interoperability.query.a` with `secure-dns-smoke`. It opens one authenticated QUIC v1 connection with TLS 1.3 and `doq` ALPN, then opens one client-initiated bidirectional stream per query.

The executor requires two-octet DoQ framing, zero DNS message ID, deterministic semantic hashes, client and server FIN, and zero malformed, retried, failed, or timed-out operations. TLS proof records Go platform provenance and an honest `not-reported` acceleration mode. Other DNS transports fail closed; local evidence is diagnostic and non-publishable.
