# Rust hyper HTTP Origins

Independent Rust hyper origin package with separate HTTP/1.1 and cleartext
HTTP/2 (h2c prior knowledge) implementation manifests. Both targets are built
from the same locked source and use digest-pinned Rust 1.88.0 and Alpine 3.22.1
images. TLS/ALPN is intentionally a separate cohort.

```powershell
pwsh ./scripts/package/Build-RustHyperOriginPackage.ps1
```
