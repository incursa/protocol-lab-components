# Governance Decisions

This page records repository governance decisions for ProtocolLab Components.

## Notice File

`NOTICE.md` is not required for this repository at this time. The repository
has an Apache-2.0 license file, but no repo-local attribution text, upstream
NOTICE file, or third-party notice policy that requires a repository NOTICE
file. Add one only if future third-party notices, upstream NOTICE text, or
redistribution terms require it.

## Release Policy

Release versioning follows SemVer. Package versions advance based on the public
component package surface:

- major for breaking package manifest, entrypoint, or public behavior changes
- minor for additive compatible package capabilities
- patch for compatible corrections

Official releases are created by maintainer-controlled Git tags. Component
package artifacts may be built from tagged sources, but the repository does not
need a separate release workflow until package publication automation is
deliberately introduced.
