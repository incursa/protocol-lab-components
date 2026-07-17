# aioquic WebTransport

Package-backed aioquic 1.3.0 server for `webtransport.session-bidi-echo`.
It uses aioquic's native WebTransport-enabled HTTP/3 connection, validates one
client-initiated bidirectional stream with the deterministic 65,536-byte
payload, and echoes the exact bytes. It does not claim WebTransport datagrams,
multiple streams, WebSocket, or MASQUE.
