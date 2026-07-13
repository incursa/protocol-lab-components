# Go TLS 1.3 Target

`go-tls13@0.1.0` is a process target for exact `tls.handshake.full` and
`tls.handshake.resumed` execution. It fixes TLS 1.3, ALPN
`protocol-lab-tls`, X25519, `TLS_AES_128_GCM_SHA256`, and the public
`plab-single-leaf-p256-v1` certificate identity. One shared `crypto/tls`
configuration retains server ticket keys across connections.

The executor owns proof of the full source session, one-time ticket
consumption, accepted resumption, warmup isolation, and zero application
bytes. Every other committed TLS scenario remains unsupported.
