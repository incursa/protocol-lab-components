# Kestrel HTTP/2 RFC 8441 WebSocket origin

`org.protocol-lab.components.implementation.kestrel-http2-websocket@0.1.0`
is an independent .NET 10 Kestrel target for the six exact HTTP/2 WebSocket
identities. It accepts only TLS 1.3 with ALPN `h2`, uses
`IHttpExtendedConnectFeature` to validate the RFC 8441 binding, and parses raw
WebSocket frames on the accepted HTTP/2 stream so client masking, payload,
control-frame, ordering, and close semantics remain visible. The committed
private key is package-local test fixture material only.
