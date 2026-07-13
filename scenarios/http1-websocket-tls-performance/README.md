# HTTP/1.1 TLS WebSocket performance scenario package

`org.protocol-lab.components.scenario.http1-websocket-tls-performance@0.2.0`
authority-locks seven exact RFC 6455 TLS 1.3 identities, the existing five-ID
smoke suite, and its load profile to public ProtocolLab commit
`8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574`.

The package does not claim cleartext substitution, TLS 1.2, RFC 8441, RFC 9220,
fragmentation, WebTransport, or the generic `websocket.echo` placeholder. The
subprotocol and permessage-deflate identities are individual diagnostics using
`websocket-smoke`; they are not silently added to the canonical five-ID suite.
