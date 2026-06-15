# ProtocolLab Components Agent Instructions

This repository owns packageable ProtocolLab component wrappers and executors.
It is not the public contract source of truth; public contracts live in the
`protocol-lab` repository.

## Authority

Follow this order when working in the repository:

1. Component package manifests named `protocol-lab-package.json`.
2. Internal execution manifests named `protocol-lab.internal.json`.
3. Package entry manifests under component-local `implementations/` or
   `test-executors/` directories.
4. Shared package validation and packaging scripts under [scripts/package/](scripts/package/).
5. Component README files, root docs, and governance files.

`package.protocol-lab.json` is legacy input only. Do not add new docs that say
every component owns `package.protocol-lab.json`, and do not restore it as the
active component manifest name.

## Boundary Rules

- Keep public package metadata in `protocol-lab-package.json`.
- Keep local execution requirements, entrypoints, runtime booleans, wrapper
  commands, and runner/tool dependencies in `protocol-lab.internal.json`.
- Do not weaken public ProtocolLab contracts to fit a local component. Update
  the public contract repo only when the public contract is genuinely wrong.
- Preserve independent `packageId` and `packageVersion` values for every
  component package.
- Keep HTTP/1, HTTP/2, HTTP/3, and QUIC lanes explicit in IDs, docs, package
  manifests, and validation.
- Generated packages and local build evidence belong under ignored artifact
  paths such as `artifacts/` and `packages/`.
- Do not commit `.plabpkg`, `.nupkg`, `.snupkg`, `bin/`, `obj/`, local auth
  state, private paths, credentials, transcripts, or private operational logs.

## Change Routing

- Manifest changes should update the matching component README and package
  validation expectations when behavior changes.
- Package script changes should preserve artifact layout under
  `artifacts/packages/` and avoid one-off release behavior in component
  directories when a shared helper is appropriate.
- Component source or wrapper changes should include the narrowest practical
  package validation and, when relevant, the affected package builder smoke.
- Governance changes may update root governance files, `.github/`, and docs
  indexes, but must not change package behavior.

## Validation

Before finishing component or governance changes, run:

```powershell
pwsh ./scripts/package/Validate-ProtocolLabComponentManifests.ps1
python C:\Users\Samuel\.codex\skills\open-source-repo-maintenance\scripts\audit_repo.py C:\shared\src\incursa\protocol-lab-components --profile incursa --format markdown
git diff --check
```

When package builders change, also run the affected wrapper in
`scripts/package/` and confirm it creates a `.plabpkg` under
`artifacts/packages/`.
