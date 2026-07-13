# Go TLS 1.3 Mutual Authentication Executor

`go-tls13-mtls-executor@0.1.0` implements only
`tls.handshake.mutual-auth`. It sends exactly the canonical client leaf (no
trust anchor), authenticates the existing canonical server leaf, and fails
closed unless TLS 1.3, X25519, `TLS_AES_128_GCM_SHA256`, and ALPN
`protocol-lab-tls` are observed on a fresh non-resumed connection.

The executor records both certificate identities in `peer-auth-proof.json`,
preserves exact executor/load-generator identities, and emits explicit
`unsupported` evidence for every other committed TLS scenario identity.
