# rustls TLS 1.3 Early Data Executor

This package supports exactly `tls.early-data.accepted` and
`tls.early-data.rejected` under `tls-diagnostic`. It provisions one unmeasured
source session, proves a single-use ticket, offers the exact 1,024-byte
repeated-`0x5A` payload, and distinguishes explicit acceptance from rejection.

On rejection, the executor retries the same payload exactly once after the
resumed handshake. It records exact TLS 1.3, AES-128-GCM, X25519, ALPN,
certificate identity, session state, payload hash, outcome, and raw artifacts.
Every other committed TLS identity is recognized as unsupported; unknown IDs
fail separately and no scenario is substituted.
