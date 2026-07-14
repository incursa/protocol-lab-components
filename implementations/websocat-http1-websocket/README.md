# websocat HTTP/1.1 WebSocket diagnostic origin

`org.protocol-lab.components.implementation.websocat-http1-websocket@0.1.0`
wraps the upstream websocat 1.14.1 mirror server for four exact cleartext
RFC 6455 scenarios: Upgrade, control frames, text echo, and close. Binary echo
is explicitly unsupported because mirror mode does not preserve the exact
binary frame opcode. The package is a diagnostic baseline only and must be
excluded from primary performance rankings because upstream documents old
dependencies, backpressure/socket-lifecycle limitations, and no RFC 8441 or
RFC 9220 support.

The release binary is pinned by SHA-256 and built into a digest-pinned Alpine
image. No ProtocolLab application server code is added.
