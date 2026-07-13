# Go HTTP/1.1 cleartext WebSocket origin

`org.protocol-lab.components.implementation.go-http1-websocket@0.1.0` is an
independent standard-library origin for the five exact cleartext RFC 6455
identities. It validates the opening handshake, requires client masking,
rejects fragmentation/extensions/subprotocols, and serves only the canonical
text, binary, ping/pong, and close semantics.
