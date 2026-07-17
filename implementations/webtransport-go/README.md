# webtransport-go

Package-backed `webtransport-go` v0.11.1 server for `webtransport.session-bidi-echo`.
It accepts WebTransport Extended CONNECT at `/webtransport/echo`, accepts one
client-initiated bidirectional stream, verifies the deterministic 65,536-byte
payload, and echoes the exact bytes. It does not claim WebTransport datagrams,
multiple streams, WebSocket, or MASQUE.
