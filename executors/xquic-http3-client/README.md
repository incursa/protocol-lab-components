# XQUIC HTTP/3 client executor

This diagnostic executor runs the client from the same digest-pinned upstream
XQUIC interop image used by the `xquic-http3` target. It validates negotiated
`h3` plus response status 200 for
`http3.external.peer-characterization`.

The wrapper intentionally does not turn XQUIC's current missing response FIN
into a payload success. It records `responseCompletionWarning=true`, retains
the full client log, and emits no canonical payload or benchmark claim.

Version `0.1.2` consumes the runner's target and artifact bindings, rewrites
loopback targets to the Docker host gateway for its nested client container,
and emits the standard HTTP executor parser record for its single diagnostic
request. The record retains the response-completion warning and makes no
payload or latency claim.

```powershell
pwsh ./scripts/package/Build-XquicHttp3ClientPackage.ps1
pwsh ./executors/xquic-http3-client/execute.ps1 -PlanOnly
```
