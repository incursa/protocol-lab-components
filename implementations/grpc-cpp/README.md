# gRPC C++ target

This package builds the canonical ProtocolLab gRPC service with Debian's pinned
gRPC C++ and protobuf packages. The synchronous adapter is generated bindings
plus exact service glue; no incompatible upstream benchmark service is used.

Version 0.1.2 exposes unary echo plus server-, client-, and bidirectional-
streaming echo alongside the proven deadline and client-cancellation rows.
Its standard `application/grpc` response media type is admitted by executor
generation 0.4.2. The encoded trailers-only message and the remaining breadth
methods stay explicitly unclaimed until separately proven.
