<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./assets/brand/protocol-lab-readme-header-white.svg">
    <img src="./assets/brand/protocol-lab-readme-header.svg" width="430" alt="ProtocolLab">
  </picture>
</p>

# ProtocolLab Components

This repository owns ProtocolLab components that are useful outside the core public contract repo:

- non-Incursa implementation adapters
- reference implementation wrappers
- smoke and compatibility test executors
- shared toolchain pins used to build those packages
- package scripts and manifest templates

The default ownership model is a component monorepo. Kestrel HTTP/1, Kestrel HTTP/2, Caddy HTTP/1, and small alternate executors should not each become a new repository just because they produce separate ProtocolLab packages. They share package conventions, release plumbing, validation scripts, and usually the same maintainers.

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
  caddy-http2/
  caddy-http3/
  nginx-http1/
  nginx-http2/
  apache-http1/
  apache-http2/
  nginx-http3/
  kestrel-http3/
  aioquic-http3/
  quiche-http3/
  ngtcp2-http3/
  quic-go-http3/
  quic-go-raw/
  quinn-raw/
  s2n-quic-raw/
  picoquic-raw/
  aioquic-raw/
  quiche-raw/
executors/
  http1-reference/
  http1-go-smoke/
  go-http1-executor/
  quic-go-raw-load/
  curl-http3-client/
  h3spec-http3-qpack/
  aioquic-rfc9220-websocket/
scenarios/
  raw-quic-transport/
  h3spec-http3-qpack/
  http3-peer-characterization/
  aioquic-rfc9220-websocket/
toolchains/
scripts/
  package/
templates/
```

`implementations/` contains runnable server/client wrappers that expose a ProtocolLab implementation package.

`executors/` contains test-executor packages. These may be reference executors, smoke executors, or compatibility checks. Executors are not fallback implementations; package consumers should select them explicitly.

`scenarios/` contains scenario-pack packages. Scenario packs publish scenario and suite manifests without carrying implementation or load-generator payloads.
Scenario packs may also carry package-relative specification documents,
requirements, catalogs, scenario mappings, and named coverage profiles. Their
presence declares mapping inputs only; it does not declare an implementation
outcome or conformance result.

`toolchains/` pins shared build inputs such as .NET SDK versions, Go versions, container base images, and external binary versions.

`scripts/package/` contains shared package validation and packaging helpers. Component directories may add local build scripts, but shared behavior should live here first.

`templates/` contains manifest templates for new implementation and test-executor packages.

## Package Conventions

Packageable component directories own a public `protocol-lab-package.json` file. The file must contain:

- `packageId`: stable package identity, unique in this repository
- `packageVersion`: independently advanced semantic version for that package
- `kind`: `implementation`, `test-executor`, or `scenario-pack`
- `entryManifests`: package-relative public catalog manifests
- `providedImplementations`, `providedTestExecutors`, or `providedScenarios`: selected public component IDs, protocols, and scenario/test coverage

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
- `org.protocol-lab.components.implementation.caddy-http2`
- `org.protocol-lab.components.implementation.caddy-http3`
- `org.protocol-lab.components.implementation.nginx-http1`
- `org.protocol-lab.components.implementation.nginx-http2`
- `org.protocol-lab.components.implementation.apache-http1`
- `org.protocol-lab.components.implementation.apache-http2`
- `org.protocol-lab.components.implementation.nginx-http3`
- `org.protocol-lab.components.implementation.kestrel-http3`
- `org.protocol-lab.components.implementation.aioquic-http3`
- `org.protocol-lab.components.implementation.quiche-http3`
- `org.protocol-lab.components.implementation.ngtcp2-http3`
- `org.protocol-lab.components.implementation.quic-go-http3`
- `org.protocol-lab.components.implementation.quic-go-raw`
- `org.protocol-lab.components.implementation.quinn-raw`
- `org.protocol-lab.components.implementation.s2n-quic-raw`
- `org.protocol-lab.components.implementation.picoquic-raw`
- `org.protocol-lab.components.implementation.aioquic-raw`
- `org.protocol-lab.components.implementation.quiche-raw`

Test-executor packages:

- `org.protocol-lab.components.executor.http1-reference`
- `org.protocol-lab.components.executor.http1-go-smoke`
- `org.protocol-lab.components.executor.go-http1-executor`
- `org.protocol-lab.components.executor.quic-go-raw-load`
- `org.protocol-lab.components.executor.curl-http3-client`
- `org.protocol-lab.components.executor.h3spec-http3-qpack`
- `org.protocol-lab.components.executor.aioquic-rfc9220-websocket`

Scenario-pack packages:

- `org.protocol-lab.components.scenario.raw-quic-transport`
- `org.protocol-lab.components.scenario.h3spec-http3-qpack`
- `org.protocol-lab.components.scenario.http3-peer-characterization`
- `org.protocol-lab.components.scenario.aioquic-rfc9220-websocket`

Kestrel packages are intentionally lane scoped. Keep HTTP/1, HTTP/2, and HTTP/3 as separate packages so controller inventory can select exact protocol behavior and report unsupported cells explicitly.

Caddy packages follow the same lane split. `caddy-http1`, `caddy-http2`, and
`caddy-http3` are separate packages so support is never inferred across
protocols. `caddy-http2` is specifically the h2c prior-knowledge variant;
TLS/ALPN is not implied.

nginx packages follow the same lane split. `nginx-http1`, `nginx-http2`, and
`nginx-http3` are separate packages. `nginx-http2` proves
`--with-http_v2_module` and exercises h2c prior knowledge; `nginx-http3` proves
`--with-http_v3_module` before serving.

Apache packages are lane scoped as `apache-http1` and `apache-http2`. They use
an unmodified digest-pinned upstream container with config/static fixtures;
the HTTP/2 h2c variant is executor-backed, while its separate TLS/ALPN variant
remains validation-unavailable until a compatible exact executor exists.

Incursa raw QUIC implementation packages remain implementation-owned by `quic-dotnet`. This repository packages the reusable raw QUIC scenario and executor pieces so controller jobs do not have to source them from local `protocol-lab-internal` scripts. The `quic-go-raw` package is a separate ecosystem target package and initially advertises only `quic.transport.stream-throughput.1mb` and `quic.transport.multiplex.100x64kb`.

The h3spec/QPACK and RFC9220 WebSocket scenario packs are declarative controller selection packs. They bind the focused suites to `h3spec-http3-qpack` and `aioquic-rfc9220-websocket` respectively so live package-backed jobs do not inherit unrelated managed HTTP/3 load suites.

The HTTP/3 peer characterization scenario pack is diagnostic. It gives external peer wrappers such as quiche and ngtcp2 a package-backed scenario identity without promoting validation-failed official `http3.payload.*` rows.

## Brand Assets

This repository reuses the official ProtocolLab identity for repository
presentation. The public
[`incursa/protocol-lab`](https://github.com/incursa/protocol-lab) repository is
the source of truth for the brand, its usage guidance, and its licensing
terms. The files under [`assets/brand/`](assets/brand/) are not component
package payloads and do not change package behavior.

The repository's code and documentation are licensed under Apache-2.0. The
ProtocolLab name, Measurement Gate logo and symbol, and files under
`assets/brand/` are separate proprietary brand assets and are not licensed
under Apache-2.0. See the local [brand asset license boundary](assets/brand/LICENSE.md).

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE).
