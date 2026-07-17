# gRPC over HTTP/2 performance scenario package

Version 0.4.2 carries all committed gRPC/H2 identities: unary echo, empty, fixed-metadata, gzip, one-mebibyte, server/client/bidirectional streaming, trailers-only status, deadline exceeded, client cancellation, and new-channel unary echo. It pairs with the executor generation that accepts the protobuf-equivalent `application/grpc` and `application/grpc+proto` media types. It includes the performance, breadth, terminal-outcome, and new-channel suites; smoke, diagnostic, and channel-churn profiles; and the committed gRPC service v2 fixture from ProtocolLab commit `8fb5110e7a811bcaad6fd88ba6eaeff869e3659a`. `authority-lock.json` pins every copied file and records the canonical service-contract digest.

The package does not itself claim executor or target support. Expected nonzero status, cancellation, deadline, and channel-lifecycle semantics remain distinct scenario identities and may not substitute for one another.
