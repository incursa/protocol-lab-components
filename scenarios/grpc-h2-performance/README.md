# gRPC over HTTP/2 performance scenario package

Version 0.2.0 carries the exact unary echo, empty, fixed-metadata, gzip, and one-mebibyte public scenarios; the performance and contract-breadth smoke suites; the smoke and diagnostic profiles; and the committed gRPC service v2 fixture from ProtocolLab commit `8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574`. `authority-lock.json` pins every copied file and records the canonical service-contract digest.

The package does not itself claim executor or target support. The contract-breadth suite also names streaming scenarios; those remain explicit unsupported cases in version 0.2.0 executor and target packages.
