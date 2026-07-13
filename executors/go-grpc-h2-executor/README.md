# Go gRPC over HTTP/2 executor

Version 0.2.0 supports `grpc.h2.unary.echo` with `grpc-h2-smoke` and the exact `grpc.h2.unary.empty`, `grpc.h2.unary.fixed-metadata`, `grpc.h2.unary.gzip`, and `grpc.h2.unary.large` identities with `grpc-h2-diagnostic`. It opens an authenticated TLS 1.3 connection with mutual ALPN `h2`, creates one HTTP/2 channel, performs an unmeasured validity operation, then measures one unary operation on that same channel.

The executor directly observes HTTP/2 status, media type, response trailers, `grpc-status`, the authenticated leaf/SPKI hashes, and all deterministic payload/protobuf/gRPC-frame byte scopes. Gzip is compared after decompression because the public contract deliberately does not make encoder-specific wire bytes comparable. Fixed metadata validates the exact ASCII value and decoded binary bytes. It emits normalized JSON plus the request frame, response frame, peer certificate, and encoder provenance. The invoking runner remains responsible for preserving stdout and stderr.

Every other committed gRPC/H2 scenario ID is explicit `protocol-lab.unsupported.v1` with exit code 3. Unknown selectors exit 2. The executor never substitutes unary echo for another selector.
