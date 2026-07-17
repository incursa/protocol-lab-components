# webtransport-go

Package-backed `webtransport-go` v0.11.1 server for
`webtransport.session-bidi-echo` and `webtransport.session-datagram-echo`.
It accepts WebTransport Extended CONNECT at `/webtransport/echo`, then either
verifies and echoes the deterministic 65,536-byte bidirectional-stream payload
or the ordered set of 32 deterministic 256-byte datagrams. It does not claim
multiple streams, unidirectional streams, WebSocket, or MASQUE.
