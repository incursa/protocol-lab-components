# H2O Experimental HTTP/3 Origin

This package builds H2O from pinned upstream commit
`edd7a120bfc4af11ac0cbebce2a43cc1f93f9af1`. H2O labels HTTP/3 experimental;
the package is therefore an experimental HTTP/3 origin and must not enter a
decision-ready origin comparison without separate evidence-policy approval.

## Coverage

- `http3.core.status`
- `http3.payload.bytes.1kb`
- `http3.payload.bytes.64kb`

The package deliberately does not claim QPACK stress, uploads, streaming, or
RFC 9220 WebSockets. It produces a short-lived self-signed certificate inside
the container for local lab validation.

## Local checks

```powershell
pwsh ./run.ps1 -PlanOnly
pwsh ./run.ps1 -ProofOnly
```
