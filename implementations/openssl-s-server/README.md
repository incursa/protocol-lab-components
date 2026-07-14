# OpenSSL `s_server` TLS endpoint

`org.protocol-lab.components.implementation.openssl-s-server@0.1.1`
source-builds the unmodified OpenSSL 3.3.0 `s_server` diagnostic executable in
a package-local Docker target. It is a TLS endpoint tool, not an application
server and not a production-origin ranking candidate.

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

The build pins the Debian base-image digest, the official OpenSSL release
archive SHA-256, the annotated tag object, and its dereferenced source commit.
The resulting target requires Docker only; it does not inspect or execute a
host-installed OpenSSL.

Use `run.ps1 -PlanOnly` or `run.sh --plan-only` for a non-starting package
smoke. Use `run.ps1 -ProofOnly` or `PLAB_PROOF_ONLY=true ./run.sh` to build the
image and prove its exact executable version without starting the endpoint.
After building the scenario, executor, and both endpoint packages into
`artifacts/tls-endpoint-docker-packages`, run
`scripts/package/Test-TlsEndpointToolDockerPackageSmoke.ps1` for the exact
extracted-package executor smoke shared with the GnuTLS target.
