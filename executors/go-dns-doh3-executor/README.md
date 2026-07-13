# Go DoH3 executor

Exact package-backed executor for the seven committed `dns.doh3.*` identities. It requires QUIC v1, TLS 1.3, ALPN `h3`, authenticated certificate hashes, exact DoH authority/path/method/media/cache binding, parsed DNS semantic parity, canonical response hashes, connection reuse, and zero malformed/retry/failure/timeout outcomes.

Other DNS transports fail closed as `unsupported`; unknown IDs and substituted executor/generator/protocol identities fail closed as configuration errors. Local output is diagnostic and non-publishable.
