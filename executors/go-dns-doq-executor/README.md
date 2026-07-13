# Go DNS-over-QUIC test executor

This package executes only `dns.doq.query.a` with `secure-dns-smoke`. It opens
one authenticated QUIC v1 connection with TLS 1.3 and `doq` ALPN, reuses that
connection, and opens one client-initiated bidirectional stream per query. It
requires the two-octet DoQ message framing, zero DNS message ID, deterministic
semantic hashes, client and server FIN, and zero malformed, retried, failed, or
timed-out operations.

All other DNS identities return explicit `unsupported`. The executor has no
DoT, DoH, classic DNS, or generic raw-QUIC fallback. Local package smoke
evidence is diagnostic and non-publishable.
