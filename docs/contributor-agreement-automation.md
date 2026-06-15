# Contributor Agreement Automation

This repository uses the Incursa contributor agreement pattern.

## Local Files

- Agreement text: [../CONTRIBUTOR-AGREEMENT.md](../CONTRIBUTOR-AGREEMENT.md)
- Workflow: [../.github/workflows/contributor-agreement.yml](../.github/workflows/contributor-agreement.yml)
- Pull request checklist: [../.github/PULL_REQUEST_TEMPLATE.md](../.github/PULL_REQUEST_TEMPLATE.md)
- Contributor instructions: [../CONTRIBUTING.md](../CONTRIBUTING.md)

The signing phrase must remain exactly:

```text
I have read the Incursa Contributor Agreement and I hereby assign my contribution rights as described.
```

## Workflow Settings

- Action: `incursa/contributor-agreement-action@v0.1.1`
- Storage repository: `incursa/contributor-agreements`
- Storage path: `signatures/incursa-contributor-agreement-v1.json`
- Storage branch: `main`
- Agreement ID: `incursa-contributor-agreement-v1`
- Required status check name: `Contributor Agreement`

## Repository Settings

`INCURSA_CONTRIBUTOR_AGREEMENTS_TOKEN` is expected to be available from the
Incursa organization secret set. Repository settings should confirm that this
repository can read that secret.

After the workflow has run on the protected branch, branch rules should require
`Contributor Agreement`, the repo validation check, and CODEOWNERS review.
Private vulnerability reporting should be enabled for the repository, with
`security@incursa.com` as the fallback contact in [`../SECURITY.md`](../SECURITY.md).
