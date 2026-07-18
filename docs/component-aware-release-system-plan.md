# Component-aware release system implementation plan

## Scope and boundaries

This plan changes component selection and future package provenance in
`protocol-lab-components`. It coordinates with, but does not take release
authority from:

- `protocol-lab`, which owns public package and contract compatibility;
- `protocol-lab-internal`, which owns immutable package admission and runner
  provenance; and
- `protocol-lab-site`, which renders immutable package provenance from public
  reports.

Existing package archives, attestations, controller records, public reports,
and site rows remain valid legacy history. The site continues to display the
existing package ID, version, and SHA-256 fields; catalog context is additive.

## Current-state audit

| Surface | Observed state | Migration response |
| --- | --- | --- |
| Component builder | Deterministic ZIP layout but embeds checkout `HEAD` and commit timestamp in package bytes | Replace embedded checkout identity with component closure identity; retain checkout commit externally. |
| Full builder/CI | `Build-AllProtocolLabComponentPackages.ps1` runs for all PR and `main` builds | Preserve it as nightly/compatibility coverage; add graph-driven dry-run selection first. |
| Package store | Rejects a SHA mismatch for an existing package ID/version | Extend metadata behavior only after additive registry compatibility tests. |
| Public site | Ingests implementation, executor, and scenario package ID/version/SHA from reports | Keep those fields stable; do not make a catalog snapshot a public evidence claim. |

## Component graph format

`release/component-graph.v1.json` is the sole authority for targeted
selection. A component lists exact repository-relative inputs in typed groups:

- `payload`: files incorporated into the package;
- `buildRecipe`: component-specific builder/wrapper files;
- `shared`: explicitly consumed shared helpers or runtime files;
- `fixtures`, `templates`, `contracts`, and `toolchains`: closure inputs with
  their expected digest/version;
- `dependsOn` and `reverseDependencies`: declared graph edges; and
- `owners`, `smoke`, and migration state.

Glob syntax is intentionally not part of the first format. A component root is
an explicit reviewed directory boundary; individual non-root inputs are exact
paths. Unknown changed paths cause a conservative fallback, never a guessed
release selection.

The graph is incrementally populated. Unmodeled package roots are represented
by `legacyFallback` and require global validation/all-build dry-run while they
are migrated. Graph validation requires one authoritative declaration before a
component can be selected or released.

## Release-intent format

`release/release-intents/*.json` records the change classification. It has one
of two outcomes:

- `release`: component IDs, requested immutable versions, changelog paths,
  approval reference, and release note; or
- `no-release`: a scoped reason why package-affecting source or release-system
  changes do not publish a package.

Intent is validated against graph selection and package manifests. A release
intent cannot include an unaffected component, reuse a version, or omit a
release note. A no-release record cannot be used by a publication command.

## Safe implementation phases

1. **Architecture and compatibility:** this ADR, graph schema, intent schema,
   catalog-index schema, and acceptance matrix.
2. **Read-only compatibility path:** graph and intent validators, closure
   digest calculation, and a selection report. Existing builds and publication
   behavior remain unchanged.
3. **Targeted tests:** prove unaffected package stability, reverse-dependency
   selection, intent enforcement, immutable version enforcement, and snapshot
   reconstruction from an index.
4. **Pilot migration:** model an HTTP/2 core cohort (scenario pack, executor,
   and Kestrel, Caddy, nginx, and Apache implementations), plus a shared-helper
   dependency. Do not republish their existing versions.
5. **CI shadow mode:** run graph selection beside the existing full build;
   report selected components and conservative fallbacks. Nightly builds all
   packages and never publishes.
6. **Registry/site additions:** add catalog snapshot and lifecycle metadata as
   additive internal/site-compatible fields; retain historical reports.
7. **Publication enablement:** only after shadow results and review are clean,
   enable an approved single-component release command with namespaced tag
   preflight. Full-repository builds never publish automatically.

## Acceptance criteria

- An unrelated committed file changes neither the declared component closure
  digest nor package bytes for a selected component.
- A payload change selects that component and all declared reverse dependents;
  a docs-only change selects no release.
- Shared, template, and contract classifications follow the ADR matrix and
  unknown inputs fail conservative rather than guessing.
- A candidate whose bytes differ from an existing `packageId + version` is
  rejected; a rollback can only move a latest pointer.
- A release candidate has exactly one approved intent, a version advance, and
  a release-note/changelog entry; unrelated components retain their versions.
- A catalog snapshot reconstructs every entry's package tuple and closure
  fields without depending on mutable registry state.
- Old manifests, old package records, and legacy public reports still validate
  and render.
- CI-like dry runs produce no publication, tag, registry mutation, or evidence
  change.

## Rollback

During migration, disabling the graph-driven CI job returns to the existing
validation/full-build workflow. No published artifact is removed or replaced.
If a released package is withdrawn, mark it yanked or superseded in registry
metadata and preserve its artifact and evidence lineage.
