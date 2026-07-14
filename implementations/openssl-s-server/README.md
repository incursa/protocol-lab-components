# OpenSSL `s_server` TLS endpoint

`org.protocol-lab.components.implementation.openssl-s-server@0.1.0` wraps the
unmodified OpenSSL 3.3.0 `s_server` diagnostic executable. It is a TLS endpoint
tool, not an application server and not a production-origin ranking candidate.

The only declared row is `tls.handshake.full`. The wrapper checks the exact
OpenSSL version, pins TLS 1.3, `TLS_AES_128_GCM_SHA256`, X25519, the canonical
ECDSA certificate, `protocol-lab-tls` ALPN, and disables caches and tickets.
The existing Go TLS executor independently validates the negotiated version,
cipher, group, ALPN, certificate DER/SPKI, non-resumption, and zero application
bytes.

Resumption, record transfer, mTLS, early data, and KeyUpdate rows remain
explicitly unsupported for the exact reasons in the entry manifest. In
particular, the upstream echo/HTTP modes are not substitutes for the
`PLABTLS1` record command protocol.

Use `run.ps1 -PlanOnly` or `run.sh --plan-only` for a non-starting package
smoke. Set `PLAB_OPENSSL_PATH` only when the exact 3.3.0 executable is not on
`PATH`.
