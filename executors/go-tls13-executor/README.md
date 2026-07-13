# Go TLS 1.3 Test Executor

`go-tls13-executor@0.1.0` executes only `tls.handshake.full` with
`tls-smoke`. It opens a fresh TCP connection, starts the primary
timer immediately before the TLS handshake, requires authenticated TLS 1.3,
ALPN `protocol-lab-tls`, the exact `plab-single-leaf-p256-v1` DER and SPKI
hashes, no resumption, and zero application data.

The primary `tlsHandshakeLatency` excludes TCP establishment. The optional
`connectionAndHandshakeLatency` begins before TCP connect. Neither boundary is
a pass/fail latency number.

Unsupported: TLS 1.2, resumption, 0-RTT, record throughput, nonzero
application data, comparison/ranking, certificate bypass, and alternate ALPN.
