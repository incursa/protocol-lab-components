# gRPC Java Netty target

This package hosts the canonical ProtocolLab gRPC service through generated
gRPC Java bindings and the Netty transport. Maven inputs and runtime versions
are pinned. No upstream benchmark service is substituted.

Version 0.1.2 exposes unary echo plus server-, client-, and bidirectional-
streaming echo alongside the proven deadline and client-cancellation rows.
Its standard `application/grpc` response media type is admitted by executor
generation 0.4.2. The percent-encoded trailers-only message and the remaining
breadth methods stay explicitly unclaimed until separately proven.
