# Go gRPC over HTTP/2 executor

This test-side executor supports only `grpc.h2.unary.echo` with `grpc-h2-smoke`. It opens an authenticated TLS 1.3 connection with mutual ALPN `h2`, creates one HTTP/2 channel, performs an unmeasured warmup RPC, then measures one unary echo on that same channel.

The executor directly observes HTTP/2 status, media type, response trailers, `grpc-status`, the authenticated leaf/SPKI hashes, and all deterministic payload/protobuf/gRPC-frame byte scopes. It emits normalized JSON plus the request frame, response frame, and peer certificate. The invoking runner remains responsible for preserving stdout and stderr.

Every other committed gRPC/H2 scenario ID is explicit unsupported. The executor never substitutes unary echo for another selector.
