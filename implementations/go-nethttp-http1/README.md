# Go net/http HTTP/1 Origin

Independent HTTP/1.1 origin implemented with the Go standard library. The
package is built as a self-contained target binary and declares only the shared
plaintext and JSON origin rows.

Build the immutable Linux package with:

```powershell
pwsh ./scripts/package/Build-GoNetHttpHttp1Package.ps1 -RuntimeIdentifier linux-x64
```
