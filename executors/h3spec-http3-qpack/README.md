# h3spec HTTP/3 and QPACK Executor

This component packages an h3spec-based ProtocolLab test executor for bounded HTTP/3 and QPACK conformance triage. It wraps h3spec v0.1.13 and emits structured JSON/Markdown artifacts using the package-local parser.

The executor does not start the target server. A controller or local operator must provide the HTTP/3 endpoint.

## Package

- Package ID: `org.protocol-lab.components.executor.h3spec-http3-qpack`
- Package version: `0.1.0`
- Executor ID: `h3spec-http3-qpack`
- Tool: h3spec `v0.1.13`
- Default wrapper image: `ubuntu:24.04`

## Local Commands

Validate component manifests from the repository root:

```powershell
pwsh ./scripts/package/Validate-ProtocolLabComponentManifests.ps1
```

Run a package-local plan-only smoke:

```powershell
pwsh ./executors/h3spec-http3-qpack/execute.ps1 -PlanOnly
```

Acquire h3spec into the component-local artifact cache:

```powershell
pwsh ./executors/h3spec-http3-qpack/Install-H3SpecTool.ps1
```

Run against a live local HTTP/3 server:

```powershell
pwsh ./executors/h3spec-http3-qpack/execute.ps1 -AcquireH3Spec -NoValidateCertificate -HostName 127.0.0.1 -Port 4433
```

Build the package artifact:

```powershell
pwsh ./scripts/package/Build-H3SpecHttp3QpackPackage.ps1
```

The package artifact is written under `artifacts/packages/`.

## Evidence Notes

Filtered h3spec runs that select no cases are classified as tooling evidence only. They are not conformance evidence. Live runs should preserve `h3spec-metadata.json`, `h3spec-results.json`, `h3spec-report.md`, stdout, and stderr together.
