# TLS 1.3 early-data component delivery

Authority: `protocol-lab@8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574`.

This component-only slice adds `org.protocol-lab.components.executor.rustls-tls13-early-data-executor@0.1.0` and independent target `org.protocol-lab.components.implementation.rustls-tls13-early-data@0.1.0`. It reuses the existing authority-locked `org.protocol-lab.components.scenario.tls13-handshake-performance@0.2.0` package.

The only supported identities are `tls.early-data.accepted` and `tls.early-data.rejected`. The executor and target fail closed for the other eight committed TLS identities as explicit `unsupported`; unknown identities remain distinct configuration errors and are never substituted.

Both identities require TLS 1.3, ALPN `protocol-lab-tls`, an authenticated package certificate, a PSK-resumed measured connection, exactly one offered 1024-byte early-data payload containing repeated `0x5A`, and payload SHA-256 `e8fb68ce4d4d002dba40c0a459d96807c96ded1c2fdefae3f56f8a0c06a4fecf`. The accepted identity requires the target to process those bytes exactly once during early data. The rejected identity requires zero early bytes processed, exactly one retry of the same bytes after the handshake, one application effect, and zero duplicate effects.

The implementation is pinned by `Cargo.lock` to `rustls@0.23.35` and `rustls-rustcrypto@0.0.2-alpha`. Package builders compile Windows x64 and Linux x64 release binaries and preserve the dependency notices and license files under `third-party-licenses/`. The package contains no forked standard library, `unsafe` implementation, or `go:linkname` dependency.

## Clean package evidence

Clean packages were built from component commit `abd4ff97fb16c0fdb82531e97542d7973c80c1c4`. All matching build attestations report clean source, are parity-eligible, and pass `Test-ProtocolLabPackageBuildAttestation.ps1 -RequireParityEligible`.

| Package artifact | SHA-256 |
| --- | --- |
| Scenario, portable | `8616f967587af620e06c8f6973f3ceb2b0988c0598f37e58dfdcbe99c75bd50a` |
| Executor, Windows x64 | `3801235aca985d900f3839df13218d30e10e2f4c3fefbfc229af8700c6457287` |
| Executor, Linux x64 | `2a0656935e59f1b94c6ba59eae9cacc23cf6883f9f3482f55c7e00047ac3d07e` |
| Target, Windows x64 | `106dc89ce88f7412c8f38b19fa409630e779673e7ad05f52f0d0aafb85b52765` |
| Target, Linux x64 | `ac2100aa2593f144b2b60b2928f31c25b004bd970dd3064a3c0daa9971b7bf6e` |

The clean packages are rooted at `artifacts/tls-early-data-clean-packages-abd4ff9`. The extracted Windows three-package smoke is rooted at `artifacts/tls-early-data-clean-extracted-smoke-abd4ff9` and produced:

| Scenario | Outcome | Completed | Failed | Timed out | Transferred bytes | Duplicate effects |
| --- | --- | ---: | ---: | ---: | ---: | ---: |
| `tls.early-data.accepted` | accepted during early data | 1 | 0 | 0 | 1,024 | 0 |
| `tls.early-data.rejected` | rejected, then retried once after handshake | 1 | 0 | 0 | 2,048 | 0 |

The rejected cell's 2,048 transferred bytes record the 1,024 offered early bytes plus the one required 1,024-byte post-handshake retry. Its semantic payload remains exactly 1,024 bytes and causes exactly one application effect. The extracted smoke also proved explicit unsupported exit/evidence for `tls.handshake.full`, `tls.handshake.resumed`, `tls.handshake.full.tls12`, `tls.handshake.full.chacha20`, `tls.handshake.mutual-auth`, `tls.key-update.diagnostic`, `tls.record.coverage`, and `tls.record.throughput`, while an unknown identity returned the distinct configuration exit.

Verification commands:

```powershell
cargo fmt --manifest-path ./implementations/rustls-tls13-early-data/source/Cargo.toml --check
cargo test --locked --manifest-path ./implementations/rustls-tls13-early-data/source/Cargo.toml
cargo clippy --locked --manifest-path ./implementations/rustls-tls13-early-data/source/Cargo.toml --all-targets -- -D warnings
cargo fmt --manifest-path ./executors/rustls-tls13-early-data-executor/source/Cargo.toml --check
cargo test --locked --manifest-path ./executors/rustls-tls13-early-data-executor/source/Cargo.toml
cargo clippy --locked --manifest-path ./executors/rustls-tls13-early-data-executor/source/Cargo.toml --all-targets -- -D warnings
pwsh ./scripts/package/Validate-ProtocolLabComponentManifests.ps1
pwsh ./scripts/package/Test-ProtocolLabPackageBuildAttestation.ps1 -PackagePath <package> -AttestationPath <attestation> -RequireParityEligible
pwsh ./scripts/package/Test-Tls13EarlyDataThreePackageSmoke.ps1 -PackageRoot ./artifacts/tls-early-data-clean-packages-abd4ff9 -ArtifactRoot ./artifacts/tls-early-data-clean-extracted-smoke-abd4ff9
```

This delivery is local diagnostic component evidence only. It makes no runner-admission, benchmark, comparison, ranking, publication, deployment, or lab-infrastructure claim.
