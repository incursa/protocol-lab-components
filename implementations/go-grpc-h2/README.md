# Go gRPC over HTTP/2 target

Version 0.3.0 is an independent origin target for five exact unary identities plus server-, client-, and bidirectional-streaming echo. It accepts only TLS 1.3 with ALPN `h2`, the canonical method paths, `application/grpc+proto`, trailers, and the scenario-specific deterministic byte scopes. Streaming methods use one RPC and one HTTP/2 stream, validate exact cardinality and client half-close, and never emulate streaming with repeated unary calls.

The package-local `contract/echo.proto` mirrors the full committed gRPC service v2 descriptor. Parity tests bind the package, service, messages, methods, streaming flags, request/response counts, behaviors, completion triggers, response policies, terminal statuses, compression profiles, and metadata profiles to the public JSON contract. Cancellation, deadline-exceeded, trailers-only status, and new-channel identities remain explicit unsupported cases in the implementation manifest.
