# Package Scripts

Shared package scripts live here so component directories do not drift into one-off release behavior.

Expected script responsibilities:

- validate `protocol-lab-package.json` and `protocol-lab.internal.json` files
- build implementation and test-executor payloads
- stage package artifacts under `artifacts/packages/`
- preserve each package's independent `packageId` and `packageVersion`
- emit public and internal manifest files alongside generated package artifacts

Component-specific build steps may live beside the component, but shared packaging behavior should be implemented here.

## Package Builders

Use the component-specific wrappers for release artifacts:

```powershell
pwsh ./scripts/package/Build-KestrelHttp2Package.ps1
pwsh ./scripts/package/Build-CaddyHttp1Package.ps1
pwsh ./scripts/package/Build-NginxHttp1Package.ps1
pwsh ./scripts/package/Build-GoHttp1ExecutorPackage.ps1
pwsh ./scripts/package/Build-KestrelHttp3Package.ps1
pwsh ./scripts/package/Build-CurlHttp3ClientPackage.ps1
pwsh ./scripts/package/Build-H3SpecHttp3QpackPackage.ps1
pwsh ./scripts/package/Build-AioquicHttp3Package.ps1
pwsh ./scripts/package/Build-QuicheHttp3Package.ps1
pwsh ./scripts/package/Build-Ngtcp2Http3Package.ps1
pwsh ./scripts/package/Build-RawQuicScenarioPackage.ps1
pwsh ./scripts/package/Build-QuicGoRawLoadPackage.ps1
```

All wrappers call `Build-ProtocolLabComponentPackage.ps1`, which reads each component's `protocol-lab-package.json` and writes a `.plabpkg` under `artifacts/packages/`.
Compiled payload wrappers may stage a runtime-specific package before compression while preserving the same package manifest layout and artifact root.
