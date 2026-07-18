# Component release metadata

This directory contains reviewed release authority for ProtocolLab Components.
It is intentionally separate from generated package indexes under `artifacts/`.

- `component-graph.v1.json` declares component dependency closures and
  selection authority.
- `release-intents/` records approved release or explicit no-release decisions.
- future catalog snapshots are immutable generated outputs committed or
  attached only by an approved release process.

Do not infer release selection from repository paths. An unknown path must
produce conservative validation/fallback, not an implicit package release.
