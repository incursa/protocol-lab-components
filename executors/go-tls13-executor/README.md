# Go TLS 1.3 Test Executor

`go-tls13-executor@0.2.0` executes `tls.handshake.full` and
`tls.handshake.resumed` with `tls-smoke`. Both modes require authenticated TLS
1.3, ALPN `protocol-lab-tls`, the exact `plab-single-leaf-p256-v1` DER and SPKI
hashes, and zero application data.

For every measured resumed operation, the executor creates a new one-entry
session cache, completes a proven full priming handshake outside the measured
window, receives a TLS 1.3 ticket, and consumes that ticket exactly once in a
handshake with `DidResume=true`. Warmup operations use separate caches and are
never reused by measurement. Rejected resumption fails validation instead of
being measured as a full handshake.

The primary `tlsHandshakeLatency` excludes TCP establishment. The optional
`connectionAndHandshakeLatency` begins before TCP connect. Neither boundary is
a pass/fail latency number.

Unsupported: TLS 1.2, 0-RTT, record throughput, nonzero
application data, comparison/ranking, certificate bypass, and alternate ALPN.
