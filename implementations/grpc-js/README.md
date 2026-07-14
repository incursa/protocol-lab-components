# grpc-js gRPC target

This package hosts the canonical ProtocolLab gRPC service with Node and
`@grpc/grpc-js`. Dependencies are locked by `package-lock.json`; the adapter is
only generated bindings and service glue. It does not claim gRPC-Web, h2c,
HTTP/3, retries, or hedging.

Nine scenarios pass the exact executor. Empty-message framing, gzip response
framing, and the literal trailers-only message remain explicit unsupported
rows because grpc-js's normal wire output differs from the current executor's
exact requirements.
