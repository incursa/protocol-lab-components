# Go TLS 1.2 Compatibility Target

`go-tls12@0.1.0` implements only `tls.handshake.full.tls12`. It pins TLS 1.2,
X25519, `TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256`, ALPN
`protocol-lab-tls`, and the canonical P-256 server leaf. Tickets and
application data are disabled. Every adjacent TLS identity is unsupported.
