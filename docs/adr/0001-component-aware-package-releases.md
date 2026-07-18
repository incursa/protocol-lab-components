# ADR 0001: Component-aware package releases

**Status:** Accepted  
**Date:** 2026-07-17  
**Owners:** ProtocolLab Components maintainers

## Context

`protocol-lab-components` is a monorepo, but each package already has an
independent `packageId` and `packageVersion`. The current production workflow
builds every package for every pull request and `main` push. Its shared package
builder also embeds the repository `HEAD` and commit timestamp in the archive.
Consequently, an unrelated checkout commit can change package bytes even when a
component's payload has not changed.

The internal package store and public evidence already retain the immutable
`packageId`, version, and SHA-256 tuple. This decision must preserve those
historical records and must not imply that every monorepo change releases every
package.

## Decision

Release authority is the explicit component graph in
[`release/component-graph.v1.json`](../../release/component-graph.v1.json),
not repository membership or inferred path heuristics.

Each component declaration names:

- its stable package identity, package root, build recipe, and package payload
  inputs;
- consumed shared helpers, fixtures, templates, toolchains/base images, and
  public-contract versions;
- declared component dependencies and reverse dependencies;
- ownership routing and optional smoke recipe; and
- its migration state.

The package identity is the digest of the declared closure: normalized payload
files, consumed shared inputs, build recipe, toolchain/base-image identifiers,
and public-contract versions. The component closure digest is embedded in a
new package provenance format. The repository checkout commit remains in the
external build attestation only. Tags, catalog snapshots, publication state,
and local artifact paths are never package-byte inputs.

The three operations remain separate:

1. **Validation** performs cheap repository-wide manifest, graph, intent, and
   compatibility checks.
2. **Build** materializes and smoke-tests selected components.
3. **Release** publishes one new immutable version only after explicit approved
   release intent and registry preflight.

## CI and release policy

| Change classification | Required work | Release behavior |
| --- | --- | --- |
| Declared component payload | Global cheap validation, component build/smoke, declared dependents | Only an approved changed component may release. |
| Documentation outside a payload | Documentation checks | Never release. |
| Shared packaging/runtime/helper | Global validation and graph-selected consumers | Release only components with changed bytes and approved intent. |
| Template | Template fixtures/tests | No component release by default. |
| Public contract/schema | Compatibility audit plus opted-in or declared incompatible consumers | Release only adopting components with approved intent. |
| Nightly | Full validate/build/smoke/reproducibility pass | Never publish. |

Until a component is explicitly represented in the graph, a change touching
its legacy build surface falls back to conservative validation/all-build dry
run. It is not silently selected by a path heuristic and cannot be released by
the component-aware path.

Release tags are namespaced by component, for example
`packages/nginx-http3/v0.1.10`. A catalog snapshot tag, for example
`catalog/2026.07.17`, points at an immutable index containing the package ID,
version, artifact SHA-256, component release tag, checkout commit, component
tree digest, build-recipe digest, contract identifiers, and relevant toolchain
or base-image digests.

Release intent is an auditable record. Every package-affecting change carries
either an approved release intent or an explicit `no-release` classification.
A release intent supplies the requested version, release note/changelog entry,
and the component(s) it authorizes. It never derives a version from a build.

Registry semantics are append-only:

- an existing `packageId + version` may only resolve to its original SHA-256;
- changed bytes require a version advance;
- an unrelated component may not receive a version change;
- rollback changes a latest pointer, not an artifact;
- deprecation, supersession, and yanking alter availability metadata while
  retaining artifacts, attestations, evidence, and audit history.

The wrapper/package version is distinct from the upstream implementation
version. External repositories may produce packages through the same contract
and are registered as external components with their own source attestation.

## Consequences

This replaces full-repository publication coupling with explicit dependency
management, while retaining repository-wide validation. It makes closure
declarations reviewable and introduces an incremental migration burden. The
initial implementation is dry-run only; it does not publish, retag, alter an
existing registry entry, or republish historical artifacts.

## Alternatives considered

- **Continue releasing every package from a full build:** rejected because it
  makes repository membership release coupling and causes unnecessary version
  churn.
- **Infer affected packages from changed paths:** rejected because shared,
  generated, and contract inputs make inference incomplete and non-auditable.
- **Use checkout commit as package identity:** rejected because unrelated
  monorepo commits change package bytes.
- **Mass republish all existing packages:** rejected because immutable package
  and evidence history must remain intact.

## Migration and verification

The compatible migration, acceptance criteria, and rollback boundaries are in
[`component-aware-release-system-plan.md`](../component-aware-release-system-plan.md).
No existing package or evidence record is modified by this ADR.
