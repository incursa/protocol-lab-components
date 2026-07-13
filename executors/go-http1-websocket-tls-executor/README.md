# Go HTTP/1.1 TLS WebSocket executor

`org.protocol-lab.components.executor.go-http1-websocket-tls-executor@0.1.0`
executes only the five exact RFC 6455 TLS smoke identities. It requires TLS
1.3, authenticated package-local certificate hashes, SNI
`websocket.plab.test`, ALPN `http/1.1`, a full non-early-data session, exact
HTTP/1.1 Upgrade proof, fresh 16-byte WebSocket keys, deterministic frames,
and a clean code-1000 close.

Cleartext, TLS 1.2, RFC 8441, RFC 9220, subprotocol, permessage-deflate,
fragmentation, WebTransport, and unknown identities fail closed. Local smoke
evidence is diagnostic and non-publishable.
