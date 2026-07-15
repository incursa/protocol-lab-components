# DNS-over-QUIC performance scenario package

Package version `0.2.0` adds the authoritative `dns.doq.interoperability.query.a` scenario and `dns-doq-interoperability-smoke` suite while preserving the strict v1 scenario and suite.

All authority material, including both TLS profiles, is byte-for-byte hash
locked to public ProtocolLab commit
`c0475b05cb80362760ac57e58ecfa1610a766c10`.

It contains declarative authority only. It does not imply executor or target
support outside the separate package identities.
