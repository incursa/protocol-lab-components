# wolfSSL TLS 1.3 Full-Handshake Target

This package source-builds wolfSSL 5.9.2 from the official stable tag and
uses its upstream example server as a narrow TLS 1.3 endpoint. The launch
contract requires TLS 1.3, `TLS13-AES128-GCM-SHA256`, X25519,
`protocol-lab-tls` ALPN, and `tls.plab.test` SNI; session tickets and client
authentication are disabled.

The package contains no prebuilt wolfSSL binary. Its Docker build downloads
the hash-pinned GPL-3.0 source archive, builds the server locally, and retains
the upstream license in the image. Only `tls.handshake.full` is declared.
