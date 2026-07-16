# Node.js node:http2 HTTP/2 Origin

Independent h2c-prior-knowledge origin using Node.js `node:http2`. The package
runs in the official Node 24.4.1 Alpine OCI image pinned by digest. TLS/ALPN is
kept out of this cleartext cohort.

```powershell
pwsh ./scripts/package/Build-NodeHttp2Package.ps1
```
