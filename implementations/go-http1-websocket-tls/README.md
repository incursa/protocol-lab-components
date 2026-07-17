# Go HTTP/1.1 TLS WebSocket origin

`org.protocol-lab.components.implementation.go-http1-websocket-tls@0.2.1`
is an independent Go standard-library origin for seven exact RFC 6455 TLS
identities, including exact `plab.echo.v1` negotiation and permessage-deflate
with both no-context-takeover parameters. It accepts only TLS 1.3 with ALPN `http/1.1`, presents the
package-local `websocket.plab.test` test certificate, disables session tickets,
and rejects noncanonical Upgrade requests, payloads, frames, and closes.

The committed private key is test fixture material only. This package is not a
production certificate-provisioning pattern and does not imply comparison or
publishability evidence.
