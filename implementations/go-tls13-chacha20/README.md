# Go TLS 1.3 ChaCha20 Target

`go-tls13-chacha20@0.1.0` implements only
`tls.handshake.full.chacha20`. The independent target remains on Go
`crypto/tls`; every completed connection proves exact ChaCha20, TLS 1.3,
X25519, ALPN, fresh state, and the canonical P-256 server identity.
