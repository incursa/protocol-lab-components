# Go TLS 1.2 Compatibility Executor

`go-tls12-executor@0.1.0` implements only `tls.handshake.full.tls12` using
`go-crypto-tls12-load@0.1.0`. It requires exact TLS 1.2, X25519,
`TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256`, ALPN `protocol-lab-tls`, and the
canonical server leaf. Fresh connections cannot offer session state and carry
no application bytes. All other committed TLS IDs are explicitly unsupported.
