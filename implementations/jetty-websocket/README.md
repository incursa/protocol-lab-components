# Jetty HTTP/1.1 WebSocket origin

`org.protocol-lab.components.implementation.jetty-websocket@0.1.0` wraps Jetty
12.1.9 with a minimal parity-tested echo endpoint using Jetty's native server
WebSocket API. The exact upstream version is pinned in Maven and the runtime
image is digest pinned.

This package selects the HTTP/1.1 Upgrade binding only. Jetty's other bindings,
TLS, extensions, and subprotocols are intentionally not claimed.
