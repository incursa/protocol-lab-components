# GnuTLS `gnutls-serv` TLS endpoint

`org.protocol-lab.components.implementation.gnutls-serv@0.1.1` source-builds
the unmodified GnuTLS 3.8.9 diagnostic server in a package-local Docker target.
It is a TLS endpoint tool, not an application server and not a
production-origin ranking candidate.

The only declared row is `tls.handshake.full`. The image checks the exact
`gnutls-serv` version and uses a priority string that leaves only TLS
1.3, `TLS_AES_128_GCM_SHA256`, X25519, and
`SIGN-ECDSA-SECP256R1-SHA256`. It also requires the canonical certificate,
fatal ALPN/SNI matching, `protocol-lab-tls`, and no tickets. The existing Go
TLS executor independently validates the observed negotiation and zero-byte
handshake semantics.

Resumption, record transfer, mTLS, early data, and KeyUpdate rows remain
explicitly unsupported for the exact reasons in the entry manifest. The
upstream echo and HTTP modes are not substitutes for the `PLABTLS1` record
command protocol.

The build pins the Debian base-image digest, the official GnuTLS release
archive SHA-256, the annotated tag object, and its dereferenced source commit.
The resulting target requires Docker only; it does not inspect or execute a
host-installed GnuTLS.

Use `run.sh --plan-only` for a non-starting package smoke. Use
`PLAB_PROOF_ONLY=true ./run.sh` to build the image and prove its exact
executable version without starting the endpoint.
After building the scenario, executor, and both endpoint packages into
`artifacts/tls-endpoint-docker-packages`, run
`scripts/package/Test-TlsEndpointToolDockerPackageSmoke.ps1` for the exact
extracted-package executor smoke shared with the OpenSSL target.
