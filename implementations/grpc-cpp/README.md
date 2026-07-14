# gRPC C++ target

This package builds the canonical ProtocolLab gRPC service with Debian's pinned
gRPC C++ and protobuf packages. The synchronous adapter is generated bindings
plus exact service glue; no incompatible upstream benchmark service is used.

Deadline and client-cancellation rows pass the exact executor. The native
runtime's standard `application/grpc` content type and encoded terminal message
do not satisfy the current executor's exact response metadata gate, so all
other methods remain implemented but explicitly unclaimed.
