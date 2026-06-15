## Scope

- [ ] Component wrapper change
- [ ] Package manifest change
- [ ] Package validation or packaging script change
- [ ] Documentation/governance change

## Boundary Check

- [ ] Public package metadata remains in `protocol-lab-package.json`.
- [ ] Execution-only metadata remains in `protocol-lab.internal.json`.
- [ ] No old guidance was added that treats `package.protocol-lab.json` as the active component manifest.
- [ ] Generated packages, local artifacts, private paths, credentials, and operational state are absent.

## Validation

- [ ] `pwsh ./scripts/package/Validate-ProtocolLabComponentManifests.ps1`
- [ ] Affected package builder smoke, if package scripts or wrapper payloads changed:

## Contributor Agreement

- [ ] I have read [CONTRIBUTOR-AGREEMENT.md](../CONTRIBUTOR-AGREEMENT.md).
- [ ] I will follow [CODE_OF_CONDUCT.md](../CODE_OF_CONDUCT.md).
