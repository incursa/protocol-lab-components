# .NET SslStream TLS 1.3 Target

This lane-scoped target exists only for `tls.handshake.full`. It negotiates
TLS 1.3 and ALPN `protocol-lab-tls`, presents
`plab-single-leaf-p256-v1`, disables TLS resumption, accepts no application
payload, and uses a fresh TCP connection for every measured handshake.

The private key under `certs/` is deterministic test-only package material. It
must not be copied to the public contract repository, emitted as evidence, or
used for any non-ProtocolLab identity.

Unsupported: TLS 1.2, resumed handshakes, 0-RTT, record throughput, client
authentication, HTTP, and publishable comparison claims.

Local start:

```powershell
pwsh ./implementations/dotnet-sslstream-tls13/run.ps1 -Port 8443
```
