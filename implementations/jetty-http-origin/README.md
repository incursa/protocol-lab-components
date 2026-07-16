# Eclipse Jetty HTTP Origins

Independent Jetty 12.0.37 origin package with separate HTTP/1.1 and cleartext
HTTP/2 (h2c prior knowledge) manifests. A small Jakarta Servlet WAR is compiled
inside the digest-pinned official Jetty JDK image; the same connector supports
HTTP/1.1 and HTTP/2 while ProtocolLab keeps the evidence cohorts separate.

```powershell
pwsh ./scripts/package/Build-JettyHttpOriginPackage.ps1
```
