# rustls TLS 1.3 Full-Handshake Target

This package is a minimal protocol-library adapter for the canonical
`tls.handshake.full` workload. It pins TLS 1.3, X25519,
`TLS_AES_128_GCM_SHA256`, `protocol-lab-tls` ALPN, the canonical SNI, and the
ProtocolLab test leaf. Session tickets and application data are disabled so a
fresh full handshake is the only admitted behavior.

The implementation is a comparable library-backed runtime row. The separate
`rustls-tls13-early-data` package remains a capability-specific target and does
not contribute evidence to this full-handshake cohort.
