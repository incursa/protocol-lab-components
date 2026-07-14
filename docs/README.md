# ProtocolLab Components Docs

This documentation supports the component package repository.

## Repository Surfaces

- [Root README](../README.md) explains the monorepo boundary, package layout,
  and current component packages.
- [Package scripts](../scripts/package/README.md) explain shared validation and
  packaging behavior.
- [Package coverage matrix](package-coverage-matrix.md) records implementation
  and executor package coverage against visible ProtocolLab comparison lanes.
- [TLS endpoint/tool feasibility](tls-endpoint-tool-feasibility.md) records the
  exact OpenSSL and GnuTLS wrapper boundary plus the repository-backed rustls
  and s2n-tls no-package decisions.
- [Contributor agreement automation](contributor-agreement-automation.md)
  records the owner setup required for the CLA workflow.
- The QUIC/HTTP/3 parity matrix lives in
  `C:\shared\src\incursa\quic-dotnet\docs\protocol-lab\quic-http3-component-parity-matrix.md`
  because `quic-dotnet` owns the Incursa support and proof story.

## Manifest Names

Active package metadata lives in component-local `protocol-lab-package.json`
files. Local execution metadata lives in paired `protocol-lab.internal.json`
files when a component has runnable payloads.

Do not add new documentation that treats `package.protocol-lab.json` as the
active component manifest name.
