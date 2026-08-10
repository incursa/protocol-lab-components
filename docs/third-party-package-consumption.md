# Third-party ProtocolLab package consumption

ProtocolLab has two deliberately separate inputs:

1. the public contract repository, which defines scenarios, schemas, run-plan
   rules, and result contracts; and
2. a component package, which supplies an implementation, test executor, or
   scenario pack together with its execution-only manifest.

A package is not a conformance result. It declares what it provides. A lab or
runner must still select compatible packages, execute the selected scenario,
retain the package identities, and report `unsupported` or `unavailable`
explicitly when a cell cannot run.

## Choose your path

- To evaluate the package contract today, use the source quickstart below.
- To consume a published bundle, first check the
  [Releases page](https://github.com/incursa/protocol-lab-components/releases).
- To author a new component, read [CONTRIBUTING.md](../CONTRIBUTING.md) and
  copy the closest manifest under [`templates/`](../templates/).
- To understand scenarios, run plans, results, and claim boundaries, use the
  public [`incursa/protocol-lab`](https://github.com/incursa/protocol-lab)
  repository.

A GitHub Actions artifact is build evidence, not a public release. If the
Releases page is empty, no downloadable release bundle has been published yet.

## Five-minute source verification

This path needs Git and PowerShell 7. It deliberately builds a declarative raw
QUIC scenario package, so Docker and protocol runtime dependencies are not
required:

```powershell
git clone https://github.com/incursa/protocol-lab-components.git
Set-Location protocol-lab-components

pwsh ./scripts/package/Validate-ProtocolLabComponentManifests.ps1

$outputRoot = Join-Path (Get-Location) 'artifacts/quickstart'
pwsh ./scripts/package/Build-RawQuicScenarioPackage.ps1 `
  -OutputRoot $outputRoot

$package = Get-ChildItem -LiteralPath $outputRoot -File `
  -Filter 'org.protocol-lab.components.scenario.raw-quic-transport.*.plabpkg' |
  Sort-Object Name |
  Select-Object -Last 1

pwsh ./scripts/package/Test-ProtocolLabPackageBuildAttestation.ps1 `
  -PackagePath $package.FullName `
  -AttestationPath ($package.FullName + '.build-attestation.json') `
  -RequireParityEligible
```

Success produces one `.plabpkg`, its matching build attestation, and a final
`Validated package build attestation` message. Run this from a clean checkout;
dirty-source packages are diagnostic-only and intentionally fail the parity
gate.

## Obtain and verify a released package

Download the `.plabpkg`, its matching `.plabpkg.build-attestation.json`, and
the release `SHA256SUMS.txt` from a ProtocolLab Components release. Verify the
archive before extracting it:

```powershell
Get-FileHash .\org.protocol-lab.components.scenario.raw-quic-transport.0.1.15.plabpkg -Algorithm SHA256
Get-Content .\SHA256SUMS.txt

pwsh .\scripts\package\Test-ProtocolLabPackageBuildAttestation.ps1 `
  -PackagePath .\org.protocol-lab.components.scenario.raw-quic-transport.0.1.15.plabpkg `
  -AttestationPath .\org.protocol-lab.components.scenario.raw-quic-transport.0.1.15.plabpkg.build-attestation.json `
  -RequireParityEligible
```

Use the exact filenames and versions from `package-index.json`; the version in
this example is illustrative. Do not replace a retained package with another
archive carrying the same `packageId` and `packageVersion`. The immutable
identity is:

```text
packageId@packageVersion#sha256
```

The attestation must identify a clean source checkout for any result that is
claimed as source/package parity. Dirty-source builds are diagnostic-only.

## Inspect the package boundary

Every ProtocolLab package must contain these root entries:

```text
protocol-lab-package.json       public package v2 manifest
protocol-lab.internal.json      execution-only manifest
package-build-provenance.json  dependency-closure provenance
```

The public manifest is the only source for package identity and advertised
component IDs. The internal manifest describes environments and entrypoints;
it must not be used to infer public protocol support. The package-relative
entry manifests are the catalog inputs that the selected runner resolves.

## Pin a run plan

Run plans must pin the package identity and hash, then select IDs supplied by
those packages. A package reference uses this exact form:

```text
packageId|packageVersion|sha256
```

For a raw QUIC run, the scenario pack, implementation package, and executor
package are separate selections. The existing raw QUIC component identities
include:

```text
org.protocol-lab.components.scenario.raw-quic-transport
org.protocol-lab.components.implementation.quic-go-raw
org.protocol-lab.components.executor.quic-go-raw-load
```

The `quic-dotnet-raw-dev` package remains owned and built by the `quic-dotnet`
repository. It is consumed as an implementation package alongside the raw
QUIC scenario and executor packages; this repository does not silently replace
it with `quic-go-raw` or infer parity across implementations.

## Build a third-party package

A third-party implementation owner should:

1. use a package ID in a namespace admitted by the target package index;
2. declare a stable semantic `packageVersion` in
   `protocol-lab-package.json`;
3. keep execution-only requirements in `protocol-lab.internal.json`;
4. run the manifest validator and package-local smoke tests;
5. build from a clean checkout so the attestation is parity-eligible; and
6. submit the package, attestation, SHA-256, public manifests, and exact
   source commit together.

The full catalog builder also exercises Docker-backed packages. For that
workflow, install Docker Desktop or Docker Engine, keep the Docker daemon
running, and ensure `docker` is on `PATH`; the builder performs this check
before starting package builds. Individual package builders may have narrower
requirements, which are declared by their internal manifest and toolchain
files.

The package builder emits deterministic archive bytes for the same declared
component closure, configuration, runtime identifier, and toolchain. A
changed package requires a new package version. The package release workflow
is intentionally manual and dry-run by default; publishing also requires a
reviewed release intent. The workflow is
[`.github/workflows/release.yml`](../.github/workflows/release.yml).

## What this does not prove

Package validation proves package shape, provenance, and declared inventory. It
does not prove that every scenario runs on every operating system, that a
third-party implementation is conformant, or that a performance comparison is
publishable. Those claims require a package-pinned execution campaign with
retained telemetry, environment identity, repetitions, and the public result
schema.
