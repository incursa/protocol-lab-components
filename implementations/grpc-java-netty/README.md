# gRPC Java Netty target

This package hosts the canonical ProtocolLab gRPC service through generated
gRPC Java bindings and the Netty transport. Maven inputs and runtime versions
are pinned. No upstream benchmark service is substituted.

Deadline and client-cancellation rows pass the exact executor. Standard gRPC
Java emits `application/grpc` and percent-encodes `grpc-message`; the current
executor requires `application/grpc+proto` and a literal terminal message, so
the remaining methods stay implemented but explicitly unclaimed.
