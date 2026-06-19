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

Inside this repository, every packageable component is still independently identifiable and independently versioned. Public package identity, package version, package-relative entry manifests, and provided implementation or test-executor IDs are declared in `protocol-lab-package.json`. Execution-only requirements and entrypoints live in `protocol-lab.internal.json` when the component has a local runnable payload.

## Layout

```text
implementations/
  kestrel-http1/
  kestrel-http2/
  caddy-http1/
  nginx-http1/
  kestrel-http3/
  aioquic-http3/
  quiche-http3/
  ngtcp2-http3/
executors/
  http1-reference/
  http1-go-smoke/
  go-http1-executor/
  curl-http3-client/
  h3spec-http3-qpack/
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

Packageable component directories own a public `protocol-lab-package.json` file. The file must contain:

- `packageId`: stable package identity, unique in this repository
- `packageVersion`: independently advanced semantic version for that package
- `kind`: `implementation` or `test-executor`
- `entryManifests`: package-relative public catalog manifests
- `providedImplementations` or `providedTestExecutors`: selected public component IDs, protocols, and scenario/test coverage

Components with local execution payloads also own `protocol-lab.internal.json`. That internal manifest contains execution environments, process or script entrypoints, and runner/tool requirements. Do not mix execution-only fields such as local commands, runtime booleans, or wrapper entrypoints back into the public package manifest.

Package IDs should use a stable dotted namespace:

- `org.protocol-lab.components.implementation.kestrel-http1`
- `org.protocol-lab.components.implementation.caddy-http1`
- `org.protocol-lab.components.executor.http1-reference`

Versioning is per package. A Caddy HTTP/1 wrapper can ship `0.2.0` while Kestrel HTTP/1 remains `0.1.0`.

Shared scripts may build all packages, but publish and release metadata must preserve each package ID and version. Do not replace per-package identity with one repository-wide package version.

## Adding A Component

1. Create a directory under `implementations/` or `executors/`.
2. Copy the closest template from `templates/`.
3. Fill in package identity, version, entry manifests, provided component IDs, and execution requirements.
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
- `org.protocol-lab.components.implementation.aioquic-http3`
- `org.protocol-lab.components.implementation.quiche-http3`
- `org.protocol-lab.components.implementation.ngtcp2-http3`

Test-executor packages:

- `org.protocol-lab.components.executor.http1-reference`
- `org.protocol-lab.components.executor.http1-go-smoke`
- `org.protocol-lab.components.executor.go-http1-executor`
- `org.protocol-lab.components.executor.curl-http3-client`
- `org.protocol-lab.components.executor.h3spec-http3-qpack`

Kestrel packages are intentionally lane scoped. Keep HTTP/1, HTTP/2, and HTTP/3 as separate packages so controller inventory can select exact protocol behavior and report unsupported cells explicitly.

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE).
