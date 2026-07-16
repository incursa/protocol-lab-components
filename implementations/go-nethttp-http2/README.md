# Go net/http HTTP/2 Origin

Independent cleartext HTTP/2 origin implemented with Go `net/http` and the
pinned `golang.org/x/net/http2` h2c adapter. HTTP/2 TLS/ALPN remains a separate
future cohort; this package declares h2c prior knowledge only.

```powershell
pwsh ./scripts/package/Build-GoNetHttpHttp2Package.ps1 -RuntimeIdentifier linux-x64
```
