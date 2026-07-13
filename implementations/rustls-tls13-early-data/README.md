# rustls TLS 1.3 Early Data Target

This package implements only `tls.early-data.accepted` and
`tls.early-data.rejected`. Both cells use the exact public TLS 1.3 profile,
an unmeasured source session, a stateful single-use ticket, and the deterministic
1,024-byte repeated-`0x5A` payload.

The rejected cell uses rustls' explicit early-data rejection control while
retaining PSK resumption, then admits exactly one post-handshake retry. The
target fails closed for every other committed TLS identity. The pinned pure-Rust
crypto provider is recorded as execution metadata; it is not public authority.
