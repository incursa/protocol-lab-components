# Go TLS 1.3 Mutual Authentication Target

`go-tls13-mtls@0.1.0` is the narrow target for
`tls.handshake.mutual-auth`. It pins TLS 1.3, X25519,
`TLS_AES_128_GCM_SHA256`, ALPN `protocol-lab-tls`, and the existing canonical
server certificate. The server requires a client certificate chain rooted in
the packaged client trust anchor and then pins the canonical client leaf DER
and SPKI hashes.

Session resumption, early data, application data, and every other committed
TLS scenario identity are unsupported by this target.
