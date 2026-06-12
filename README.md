# Protocol Lab Components

This repository owns Protocol Lab components that are useful outside the core public contract repo:

- non-Incursa implementation adapters
- reference implementation wrappers
- smoke and compatibility test executors
- shared toolchain pins used to build those packages
- package scripts and manifest templates

The default ownership model is a component monorepo. Kestrel HTTP/1, Kestrel HTTP/2, Caddy HTTP/1, and small alternate executors should not each become a new repository just because they produce separate Protocol Lab packages. They share package conventions, release plumbing, validation scripts, and usually the same maintainers.

Separate repositories should be created only when there is a concrete boundary that makes shared operation more expensive than useful:

- incompatible licensing or redistribution terms
- runtime isolation requirements that cannot be represented in package metadata
- external ownership with an independent review and release process
- substantially different CI, security, or release infrastructure
- a toolchain that cannot coexist with the shared component build

Inside this repository, every component is still independently identifiable and independently versioned. Package identity, package version, runtime requirements, entrypoints, and provenance are declared in each component manifest.

## Layout

```text
implementations/
  kestrel-http1/
  kestrel-http2/
  caddy-http1/
  nginx-http1/
  kestrel-http3/
executors/
  http1-reference/
  http1-go-smoke/
  go-http1-executor/
toolchains/
scripts/
  package/
templates/
```

`implementations/` contains runnable server/client wrappers that expose a Protocol Lab implementation package.

`executors/` contains test-executor packages. These may be reference executors, smoke executors, or compatibility checks. Executors are not fallback implementations; package consumers should select them explicitly.

`toolchains/` pins shared build inputs such as .NET SDK versions, Go versions, container base images, and external binary versions.

`scripts/package/` contains shared package validation and packaging helpers. Component directories may add local build scripts, but shared behavior should live here first.

`templates/` contains manifest templates for new implementation and test-executor packages.

## Package Conventions

Each component directory owns a `package.protocol-lab.json` file. The file must contain:

- `packageId`: stable package identity, unique in this repository
- `packageKind`: `implementation` or `test-executor`
- `version`: independently advanced semantic version for that package
- `component`: local component name and protocol role
- `entrypoints`: commands, containers, or scripts exposed by the package
- `requirements`: operating system, runtime, ports, protocols, and external tools
- `provenance`: source path and maintainers

Package IDs should use a stable dotted namespace:

- `org.protocol-lab.components.implementation.kestrel-http1`
- `org.protocol-lab.components.implementation.caddy-http1`
- `org.protocol-lab.components.executor.http1-reference`

Versioning is per package. A Caddy HTTP/1 wrapper can ship `0.2.0` while Kestrel HTTP/1 remains `0.1.0`.

Shared scripts may build all packages, but publish and release metadata must preserve each package ID and version. Do not replace per-package identity with one repository-wide package version.

## Adding A Component

1. Create a directory under `implementations/` or `executors/`.
2. Copy the closest template from `templates/`.
3. Fill in package identity, version, entrypoints, requirements, and provenance.
4. Add local source, wrapper scripts, Dockerfiles, or build files next to the manifest.
5. Run `pwsh ./scripts/package/Validate-ProtocolLabComponentManifests.ps1`.

Adding Kestrel HTTP/1 or Caddy HTTP/1 is a normal component addition in this repository. It does not require creating another repository.

## Current Lane Packages

Implementation packages:

- `org.protocol-lab.components.implementation.kestrel-http1`
- `org.protocol-lab.components.implementation.kestrel-http2`
- `org.protocol-lab.components.implementation.caddy-http1`
- `org.protocol-lab.components.implementation.nginx-http1`
- `org.protocol-lab.components.implementation.kestrel-http3`

Test-executor packages:

- `org.protocol-lab.components.executor.http1-reference`
- `org.protocol-lab.components.executor.http1-go-smoke`
- `org.protocol-lab.components.executor.go-http1-executor`

Kestrel packages are intentionally lane scoped. Keep HTTP/1, HTTP/2, and HTTP/3 as separate packages so controller inventory can select exact protocol behavior and report unsupported cells explicitly.
