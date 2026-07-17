# aioquic WebTransport

Package-backed aioquic 1.3.0 server for `webtransport.session-bidi-echo` and
`webtransport.session-datagram-echo`. It uses aioquic's native
WebTransport-enabled HTTP/3 connection to verify and echo either the
deterministic 65,536-byte bidirectional-stream payload or the ordered set of
32 deterministic 256-byte datagrams. It does not claim multiple streams,
unidirectional streams, WebSocket, or MASQUE.
