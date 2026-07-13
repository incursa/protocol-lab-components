# Go TLS 1.3 Target

`go-tls13@0.2.0` preserves exact full and accepted-PSK resumed handshakes and
adds deterministic TLS record transfer. A package-selected record scenario
accepts only the canonical size/direction cases, serves or validates repeated
`0x5a` application bytes, and acknowledges client-to-server payloads with
their exact SHA-256.

The process pins TLS 1.3, X25519, `TLS_AES_128_GCM_SHA256`, ALPN
`protocol-lab-tls`, and the public test certificate. Unsupported scenario
identities are never substituted with a supported handshake or transfer path.
