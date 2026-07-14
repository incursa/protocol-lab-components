# uWebSockets HTTP/1.1 WebSocket origin

`org.protocol-lab.components.implementation.uwebsockets-websocket@0.1.0`
builds upstream uWebSockets 20.79.0 at commit
`fe7c01a477b688a7743f754fee33bdd78d52ad91` with its pinned uSockets submodule
and a minimal native echo adapter.

Compression and automatic server pings are disabled. Only the five exact
cleartext HTTP/1.1 scenarios are claimed; TLS, extensions, subprotocols,
RFC 8441, and RFC 9220 remain unsupported.
