# Go TLS 1.3 Test Executor

`go-tls13-executor@0.3.2` executes `tls.handshake.full`,
`tls.handshake.resumed`, `tls.record.throughput`, and `tls.record.coverage`.
Every mode requires authenticated TLS 1.3, ALPN `protocol-lab-tls`, X25519,
`TLS_AES_128_GCM_SHA256`, and the exact public certificate DER and SPKI hashes.

Record operations verify canonical repeated-`0x5a` payload size, SHA-256,
direction, fresh full-authenticated connections, no resumption, and no early
data. Coverage executes all six cases on separate connections. The executor
observes encrypted TLS record deltas at the underlying connection but does not
claim that Go `crypto/tls` exposes or controls plaintext-to-record boundaries.

Resumed operations retain the v0.2 one-entry session-cache proof and fail if a
ticket is not accepted exactly once. Unsupported TLS identities fail closed
with `protocol-lab.unsupported.v1` evidence.
