# HTTP/2 RFC 8441 WebSocket scenario package

`org.protocol-lab.components.scenario.http2-websocket-performance@0.1.2`
locks the six exact HTTP/2 WebSocket scenarios and three load profiles to public authority commit
`8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574`. The package adds no executable
support and makes no publishability claim.

The five core scenarios remain in `http2-websocket-performance-smoke` on
`websocket-smoke`. Ordered multi-message echo is routed only through
`http2-websocket-multi-message-diagnostic` on `diagnostic`.
