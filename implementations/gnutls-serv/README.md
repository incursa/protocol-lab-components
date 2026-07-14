# GnuTLS `gnutls-serv` TLS endpoint

`org.protocol-lab.components.implementation.gnutls-serv@0.1.0` wraps the
unmodified GnuTLS 3.8.9 diagnostic server. It is a TLS endpoint tool, not an
application server and not a production-origin ranking candidate.

The only declared row is `tls.handshake.full`. The Linux wrapper checks the
exact `gnutls-serv` version and uses a priority string that leaves only TLS
1.3, `TLS_AES_128_GCM_SHA256`, X25519, and
`SIGN-ECDSA-SECP256R1-SHA256`. It also requires the canonical certificate,
fatal ALPN/SNI matching, `protocol-lab-tls`, and no tickets. The existing Go
TLS executor independently validates the observed negotiation and zero-byte
handshake semantics.

Resumption, record transfer, mTLS, early data, and KeyUpdate rows remain
explicitly unsupported for the exact reasons in the entry manifest. The
upstream echo and HTTP modes are not substitutes for the `PLABTLS1` record
command protocol.

Use `run.sh --plan-only` for a non-starting package smoke. Set
`PLAB_GNUTLS_SERV_PATH` only when the exact 3.8.9 executable is not on `PATH`.
