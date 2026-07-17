# Jetty HTTP/2 RFC 8441 WebSocket origin

`org.protocol-lab.components.implementation.jetty-http2-websocket@0.1.0`
provides the second independent RFC 8441 ecosystem in the ProtocolLab catalog.
The adapter configures Jetty 12.1.9 for TLS 1.3, ALPN `h2`, and
`SETTINGS_ENABLE_CONNECT_PROTOCOL`, then maps `/websocket` through Jetty's
WebSocket server API. HTTP/1.1 Upgrade and HTTP/3 RFC 9220 remain separate
cohorts. The committed key is package-local test-fixture material only.
