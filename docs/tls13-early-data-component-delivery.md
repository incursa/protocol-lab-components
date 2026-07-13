# TLS 1.3 early-data component delivery

Authority: `protocol-lab@8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574`.

This component-only slice adds `org.protocol-lab.components.executor.rustls-tls13-early-data-executor@0.1.0` and independent target `org.protocol-lab.components.implementation.rustls-tls13-early-data@0.1.0`. It reuses the existing authority-locked `org.protocol-lab.components.scenario.tls13-handshake-performance@0.2.0` package.

The only supported identities are `tls.early-data.accepted` and `tls.early-data.rejected`. The executor and target fail closed for the other eight committed TLS identities as explicit `unsupported`; unknown identities remain distinct configuration errors and are never substituted.

Both identities require TLS 1.3, ALPN `protocol-lab-tls`, an authenticated package certificate, a PSK-resumed measured connection, exactly one offered 1024-byte early-data payload containing repeated `0x5A`, and payload SHA-256 `e8fb68ce4d4d002dba40c0a459d96807c96ded1c2fdefae3f56f8a0c06a4fecf`. The accepted identity requires the target to process those bytes exactly once during early data. The rejected identity requires zero early bytes processed, exactly one retry of the same bytes after the handshake, one application effect, and zero duplicate effects.

The implementation is pinned by `Cargo.lock` to `rustls@0.23.35` and `rustls-rustcrypto@0.0.2-alpha`. Package builders compile Windows x64 and Linux x64 release binaries and preserve the dependency notices and license files under `third-party-licenses/`. The package contains no forked standard library, `unsafe` implementation, or `go:linkname` dependency.

## Clean package evidence

Clean package hashes and extracted-package evidence will be recorded in a follow-up evidence-only commit after this implementation commit establishes a clean source head.

Verification commands:

```powershell
cargo fmt --manifest-path ./implementations/rustls-tls13-early-data/source/Cargo.toml --check
cargo test --locked --manifest-path ./implementations/rustls-tls13-early-data/source/Cargo.toml
cargo clippy --locked --manifest-path ./implementations/rustls-tls13-early-data/source/Cargo.toml --all-targets -- -D warnings
cargo fmt --manifest-path ./executors/rustls-tls13-early-data-executor/source/Cargo.toml --check
cargo test --locked --manifest-path ./executors/rustls-tls13-early-data-executor/source/Cargo.toml
cargo clippy --locked --manifest-path ./executors/rustls-tls13-early-data-executor/source/Cargo.toml --all-targets -- -D warnings
pwsh ./scripts/package/Validate-ProtocolLabComponentManifests.ps1
pwsh ./scripts/package/Test-Tls13EarlyDataThreePackageSmoke.ps1
```

This delivery is local diagnostic component evidence only. It makes no runner-admission, benchmark, comparison, ranking, publication, deployment, or lab-infrastructure claim.
