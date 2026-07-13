# Go gRPC over HTTP/2 target

Version 0.4.0 is an independent origin target for every committed gRPC/H2 identity. It accepts only TLS 1.3 with ALPN `h2`, canonical method paths, `application/grpc+proto`, and scenario-specific deterministic byte scopes. It preserves the unary and true streaming paths and adds exact trailers-only INVALID_ARGUMENT, bounded deadline withholding, ready-metadata cancellation, and the unary endpoint used by executor-proven channel churn.

The package-local `contract/echo.proto` mirrors the full committed gRPC service v2 descriptor. Parity tests bind the package, service, messages, methods, streaming flags, request/response counts, behaviors, completion triggers, response policies, terminal statuses, compression profiles, and metadata profiles to the public JSON contract. The target does not itself claim client-side cancellation, deadline, or fresh-channel proof; those facts remain executor evidence.
