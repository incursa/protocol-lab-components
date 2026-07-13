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

Use the full package build orchestrator for CI artifacts and release handoff:

```powershell
pwsh ./scripts/package/Build-AllProtocolLabComponentPackages.ps1 -Clean
```

The orchestrator validates all component manifests, builds every packageable
component wrapper, inspects each archive for the required public/internal
manifest split, and writes these outputs under `artifacts/packages/`:

- `.plabpkg` files for each package artifact
- `.plabpkg.build-attestation.json` files for each package artifact
- `package-index.json`
- `package-index.md`
- `SHA256SUMS.txt`
- `package-validation-summary.json`
- `package-validation-summary.md`

Package production fails unless every package has exactly one valid,
parity-eligible clean-source build attestation. The package index and validation
summary record each attestation artifact, its SHA-256, and source commit.

Use the component-specific wrappers when iterating on one package:

```powershell
pwsh ./scripts/package/Build-KestrelHttp2Package.ps1
pwsh ./scripts/package/Build-CaddyHttp1Package.ps1
pwsh ./scripts/package/Build-CaddyHttp3Package.ps1
pwsh ./scripts/package/Build-NginxHttp1Package.ps1
pwsh ./scripts/package/Build-GoHttp1ExecutorPackage.ps1
pwsh ./scripts/package/Build-GoHttp1WebSocketImplementationPackage.ps1
pwsh ./scripts/package/Build-GoHttp1WebSocketExecutorPackage.ps1
pwsh ./scripts/package/Build-Http1WebSocketCleartextScenarioPackage.ps1
pwsh ./scripts/package/Build-KestrelHttp3Package.ps1
pwsh ./scripts/package/Build-CurlHttp3ClientPackage.ps1
pwsh ./scripts/package/Build-H3SpecHttp3QpackPackage.ps1
pwsh ./scripts/package/Build-AioquicRfc9220WebSocketPackage.ps1
pwsh ./scripts/package/Build-AioquicHttp3Package.ps1
pwsh ./scripts/package/Build-QuicheHttp3Package.ps1
pwsh ./scripts/package/Build-Ngtcp2Http3Package.ps1
pwsh ./scripts/package/Build-AioquicRfc9220WebSocketScenarioPackage.ps1
pwsh ./scripts/package/Build-H3SpecHttp3QpackScenarioPackage.ps1
pwsh ./scripts/package/Build-Http3PeerCharacterizationScenarioPackage.ps1
pwsh ./scripts/package/Build-RawQuicScenarioPackage.ps1
pwsh ./scripts/package/Build-QuicGoRawLoadPackage.ps1
```

Exercise the extracted Windows three-package HTTP/1.1 cleartext WebSocket cell,
all five exact supported identities, every adjacent explicit unsupported
identity, and the unknown-ID exit path with:

```powershell
pwsh ./scripts/package/Test-Http1WebSocketThreePackageSmoke.ps1
```

All wrappers call `Build-ProtocolLabComponentPackage.ps1`, which reads each component's `protocol-lab-package.json` and writes a `.plabpkg` under `artifacts/packages/`.
Compiled payload wrappers may stage a runtime-specific package before compression while preserving the same package manifest layout and artifact root.

The shared builder also writes `<package>.build-attestation.json` with the exact
source repository and commit, dirty state, build configuration, runtime/tool
versions, package identity, SHA-256, and materialization path. Clean source is
required by default. `-AllowDirtySource` exists only for diagnostic iteration;
such attestations set `parityEligible=false` and cannot support source/package
parity or publication.

The same source and build identity is embedded as
`package-build-provenance.json` inside the package so a worker can carry the
source lineage into materialization and parity evidence. The external
attestation adds the artifact SHA-256 and local materialization path, which
cannot be embedded without creating a self-referential hash.

Validate a retained artifact and its attestation with:

```powershell
pwsh ./scripts/package/Test-ProtocolLabPackageBuildAttestation.ps1 `
  -PackagePath <path-to.plabpkg> `
  -AttestationPath <path-to.plabpkg.build-attestation.json> `
  -RequireParityEligible
```

The shared builder writes archive entries in stable ordinal order, uses a fixed
ZIP timestamp and encoding, and bases embedded provenance time on the source
commit. For the same clean commit, configuration, runtime identifier, and
recorded toolchain, rebuilding produces byte-identical package bytes. Verify
that invariant, including the same-root immutable no-op, with:

```powershell
pwsh ./scripts/package/Test-ProtocolLabPackageReproducibility.ps1 `
  -ComponentPath implementations/kestrel-http2 `
  -BuildConfiguration Release `
  -RuntimeIdentifier win-x64
```

The external attestation timestamp is intentionally not embedded in the archive
and therefore does not affect package SHA-256.

## Versioning Policy

Package versions come from each component's `protocol-lab-package.json`
`packageVersion` value. Build and CI scripts must not synthesize timestamp
versions by default. Timestamp, run-number, or prerelease version stamping can be
added later only as an explicit release policy change.

The package identity tuple is:

- `packageId`
- `packageVersion`
- package artifact SHA-256 hash

Once a package artifact is uploaded to a package inventory or controller, that
identity tuple is immutable. Changing package contents requires a new
`packageVersion`; rebuilding a previously uploaded `packageId` and
`packageVersion` with different bytes must be treated as a different local build
that is not a replacement for the uploaded artifact.

The shared builder enforces the same rule locally: it refuses to replace an
existing package path when the candidate SHA-256 differs. Use an empty output
root for an independent reproducibility experiment or increment the package
version for changed contents.
