# gRPC over HTTP/2 performance scenario package

This package carries the exact `grpc.h2.unary.echo`, `grpc-h2-performance-smoke`, and `grpc-h2-smoke` public artifacts from ProtocolLab commit `8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574`, plus the committed gRPC service v2 fixture. `authority-lock.json` pins every copied file and records the canonical service-contract digest.

The package does not claim executor or target support. The committed smoke suite also names streaming scenarios; those remain explicit unsupported cases in the initial executor and target packages.
