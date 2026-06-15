# Contributing

Contributions are welcome when they preserve the component-package boundary and
keep public package metadata separate from execution-only runner details.

## Before You Open A Pull Request

- Read [CONTRIBUTOR-AGREEMENT.md](CONTRIBUTOR-AGREEMENT.md).
- Follow [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
- Read [README.md](README.md), [scripts/package/README.md](scripts/package/README.md),
  and [docs/README.md](docs/README.md).
- Keep changes focused on component wrappers, package metadata, manifests,
  validation, packaging scripts, documentation, or governance files.
- Verify that public component metadata stays in `protocol-lab-package.json`.
- Verify that local execution requirements and entrypoints stay in
  `protocol-lab.internal.json`.

## What To Include

- Component changes should update the matching package manifest, entry
  manifest, README, and package validation expectations.
- Package script changes should explain the artifact layout and preserve each
  package's independent `packageId` and `packageVersion`.
- Protocol coverage changes should keep HTTP/1, HTTP/2, HTTP/3, and QUIC
  lanes explicit in package IDs, supported protocols, scenario IDs, and docs.
- Documentation changes should mention the active manifest names:
  `protocol-lab-package.json` for public package metadata and
  `protocol-lab.internal.json` for local execution metadata.

## What Not To Do

- Do not restore old guidance that says every component owns
  `package.protocol-lab.json`.
- Do not mix execution-only fields such as wrapper commands, local runtime
  booleans, or runner entrypoints into `protocol-lab-package.json`.
- Do not replace per-package identity with one repository-wide package version.
- Do not include generated packages, build outputs, local artifacts, private
  workspace paths, credentials, secrets, private service URLs, or private
  operational state.
- Do not claim benchmark authority, hosted execution, certification, or
  production support from component package metadata alone.

## Contributor Agreement

Pull requests are checked by the `Contributor Agreement` workflow. If the
workflow asks you to sign, read [CONTRIBUTOR-AGREEMENT.md](CONTRIBUTOR-AGREEMENT.md)
and comment exactly:

```text
I have read the Incursa Contributor Agreement and I hereby assign my contribution rights as described.
```

The workflow records signatures outside this repository. Maintainers configure
the required secret and branch status check through GitHub repository or
organization settings.

## Validation

Run the focused package validation before review:

```powershell
pwsh ./scripts/package/Validate-ProtocolLabComponentManifests.ps1
```

When changing package builders, also run the affected wrapper under
`scripts/package/` and confirm the generated `.plabpkg` remains under
`artifacts/packages/`.

## Style

- Use clear component and protocol names in docs and package metadata.
- Keep package README files concrete about what the component provides and what
  it does not provide.
- Prefer shared package helpers over one-off component release behavior.
