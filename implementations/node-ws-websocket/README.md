# Node ws HTTP/1.1 WebSocket origin

`org.protocol-lab.components.implementation.node-ws-websocket@0.1.0` wraps
upstream `ws` 8.21.0 with a minimal parity-tested echo adapter. The adapter
preserves text versus binary frames and leaves RFC 6455 ping/pong and close
handling to `ws`; permessage-deflate is deliberately disabled.

Only the five exact cleartext HTTP/1.1 scenarios are claimed. TLS,
subprotocols, extensions, RFC 8441, and RFC 9220 remain unsupported.
