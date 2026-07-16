# XQUIC HTTP/3 client executor

This diagnostic executor runs the client from the same digest-pinned upstream
XQUIC interop image used by the `xquic-http3` target. It validates negotiated
`h3` plus response status 200 for
`http3.external.peer-characterization`.

The wrapper intentionally does not turn XQUIC's current missing response FIN
into a payload success. It records `responseCompletionWarning=true`, retains
the full client log, and emits no canonical payload or benchmark claim.

```powershell
pwsh ./scripts/package/Build-XquicHttp3ClientPackage.ps1
pwsh ./executors/xquic-http3-client/execute.ps1 -PlanOnly
```
