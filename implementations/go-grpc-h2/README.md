# Go gRPC over HTTP/2 unary target

Version 0.2.0 is an independent origin target for five exact unary identities: baseline echo, empty proto3 message, fixed metadata, gzip semantic echo, and one-mebibyte echo. It accepts only TLS 1.3 with ALPN `h2`, the canonical method paths, `application/grpc+proto`, trailers, and the scenario-specific deterministic byte scopes.

The package-local `contract/echo.proto` mirrors the full committed gRPC service v2 descriptor. Parity tests bind the package, service, messages, methods, streaming flags, terminal statuses, compression profiles, and metadata profiles to the public JSON contract. All non-unary-success committed gRPC/H2 identities remain explicit unsupported cases in the implementation manifest.
