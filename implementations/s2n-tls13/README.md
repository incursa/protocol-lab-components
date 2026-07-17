# s2n-tls TLS 1.3 Full-Handshake Target

This package source-builds s2n-tls 1.7.5 and links a minimal C adapter against
the resulting static library. The adapter accepts only the canonical
`tls.handshake.full` scenario and validates TLS 1.3,
`TLS_AES_128_GCM_SHA256`, X25519, `protocol-lab-tls` ALPN, and
`tls.plab.test` SNI after every handshake. Session tickets and application
data are disabled.

The adapter is deliberately smaller than the upstream `s2nd` testing utility:
it exists only to host the canonical fixture as a comparable library-backed
runtime. The source tag, commit, archive hash, base image digest, and linked
OpenSSL dependency are retained in package metadata.
