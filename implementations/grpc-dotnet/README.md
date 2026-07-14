# grpc-dotnet gRPC target

This package hosts the canonical `protocollab.performance.v1.EchoService` with
ASP.NET Core and grpc-dotnet. It preserves the runtime identity and uses only
generated bindings plus service glue. TLS is restricted to TLS 1.3 and ALPN
`h2`; gRPC-Web, h2c, HTTP/3, retries, and hedging are not claimed.

Build with `pwsh ../../scripts/package/Build-GrpcDotNetPackage.ps1` and run
`pwsh ./run.ps1 -ProofOnly` for the pinned runtime proof.
