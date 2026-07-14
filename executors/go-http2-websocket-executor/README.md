# Go HTTP/2 RFC 8441 WebSocket executor

`go-http2-websocket-executor@0.2.0` implements the six exact public RFC 8441
identities with `go-x-net-http2-websocket-load@0.2.0`. It uses the pinned
`golang.org/x/net/http2@v0.57.0` raw Framer and HPACK APIs so the executor can
prove the server setting, pseudo-headers, response headers, DATA frames, client
masking, payloads, ordering, control frames, and close behavior without an API
that hides the binding. All HTTP/1.1 and HTTP/3 WebSocket IDs are explicit
unsupported outcomes. Evidence is local, diagnostic, and non-publishable.

The five core identities use the exact `websocket-smoke` capacity and timing:
one connection, one concurrent operation, one stream, one-second warmup, and a
five-second measured window. The ordered multi-message identity uses the
public `diagnostic` profile on one authenticated TLS 1.3 + HTTP/2 connection:
eight concurrently active Extended CONNECT streams, one worker per stream,
and exact 100-message ordered operations for a one-second warmup and ten-second
measured window followed by one second of cooldown. Configured capacity and
observed active connections, streams, and concurrency are reported separately.
