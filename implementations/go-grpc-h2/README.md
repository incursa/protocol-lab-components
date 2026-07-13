# Go gRPC over HTTP/2 unary target

This is an independent origin target for `grpc.h2.unary.echo`. It accepts only TLS 1.3 with ALPN `h2`, the canonical RPC path, `application/grpc+proto`, trailers, and the exact 136-byte identity frame containing the 131-byte protobuf representation of 128 `G` octets.

The package-local `contract/echo.proto` mirrors the full committed gRPC service v2 descriptor so parity tests can detect drift, but version 0.1.0 executes only unary echo. All other committed gRPC/H2 scenario IDs are explicit unsupported cases.
