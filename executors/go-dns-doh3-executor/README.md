# Go DoH3 executor

Package version `0.2.1` executes the seven strict `dns.doh3.*` identities plus authoritative `dns.doh3.interoperability.query.a`. It requires QUIC v1, TLS 1.3, ALPN `h3`, authenticated certificate proof, exact DoH authority/path/method/media binding, parsed DNS semantic parity, canonical response hashes, connection reuse, zero malformed, retry, failure, and timeout outcomes, and a protocol variant bound to the selected strict or interoperability scenario.

TLS proof records Go platform provenance and an honest `not-reported` acceleration mode. Other DNS transports fail closed; unknown or substituted identities fail as configuration errors. Local output is diagnostic and non-publishable.
