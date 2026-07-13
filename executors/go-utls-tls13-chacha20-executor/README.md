# Go uTLS TLS 1.3 ChaCha20 Executor

`go-utls-tls13-chacha20-executor@0.1.0` implements only
`tls.handshake.full.chacha20` using `go-utls-tls13-chacha20-load@0.1.0` and
the pinned `github.com/refraction-networking/utls@v1.8.2` module.

The executor applies a custom ClientHelloSpec that offers only TLS 1.3,
`TLS_CHACHA20_POLY1305_SHA256`, X25519, the P-256 ECDSA signature algorithm,
and ALPN `protocol-lab-tls`. It offers no ticket, PSK, or early-data state. The
raw ClientHello proof and negotiated state must agree before evidence passes.
The target remains an independent Go `crypto/tls` server. All adjacent TLS IDs
are explicitly unsupported.
