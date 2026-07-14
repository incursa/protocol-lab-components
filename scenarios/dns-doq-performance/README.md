# DNS-over-QUIC performance scenario package

Package version `0.1.1` declares the bundled `dns-doq-performance-smoke` suite without changing public authority bytes.

This package authority-locks `dns.doq.query.a`,
`dns-doq-performance-smoke`, `secure-dns-smoke`, the canonical A-query wire
fixture, and the secure-DNS TLS profile to public ProtocolLab commit
`8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574`.

It contains declarative authority only. It does not imply executor or target
support outside the separate package identities.
