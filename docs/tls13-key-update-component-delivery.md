# TLS 1.3 KeyUpdate component delivery

Authority: `protocol-lab@8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574`.

This component-only slice adds `org.protocol-lab.components.executor.openssl-tls13-key-update-executor@0.1.0` and independent target `org.protocol-lab.components.implementation.openssl-tls13-key-update@0.1.0`. It reuses the authority-locked `org.protocol-lab.components.scenario.tls13-handshake-performance@0.2.0` package, whose lock covers `tls.key-update.diagnostic`, `tls-diagnostic`, and `plab-tls13-aes128gcm-p256-server-auth-v2`.

The only supported identity is `tls.key-update.diagnostic`. The executor and target fail closed for every other committed TLS identity as explicit `unsupported`; unknown identities remain distinct configuration errors.

The client executor calls OpenSSL's public `SSL_key_update(SSL_KEY_UPDATE_NOT_REQUESTED)` API exactly once. Its message callback proves one outgoing TLS 1.3 KeyUpdate carrying request byte zero. The independent target's message callback proves one incoming KeyUpdate and zero outgoing updates. Successful decryption of the exact 65,536-byte repeated-`0x5A` post-update payload proves the client-write/server-read traffic generation transition; the deterministic 65,536-byte response proves bidirectional connection continuity. The proof records generation numbers and event counts but never enables key logging or publishes traffic-secret material.

Windows packages use package-local OpenSSL `3.3.0` libraries. Linux packages statically link OpenSSL `3.5.7` using the pinned Alpine `3.22.2` image digest. Both packages include Apache-2.0 license material.

## Clean package evidence

Clean packages were built from component commit `e414416a0a0c11538df954beacd8ca5e87fccccf`. All matching build attestations report clean source, are parity-eligible, and pass `Test-ProtocolLabPackageBuildAttestation.ps1 -RequireParityEligible`.

| Package artifact | SHA-256 |
| --- | --- |
| Scenario, portable | `91182c53e90d7c284d2d65e9940d30e6467e6321c671136f64e5f8530a0900e5` |
| Executor, Windows x64 | `31ad801e8cd54ede18df2aaac307f0579b7427f19913049458e1dfabd5c739d5` |
| Executor, Linux x64 | `c977deec0ee120e1d6b8a0153cc13bd57a5c4beb5726a5f30b5c20643a8a5f80` |
| Target, Windows x64 | `4a6938dafcf57c428e9c66571cc63f6be41e9d65c210bccff23759ab89a4ad88` |
| Target, Linux x64 | `0b5386fa1361ea4d58bd7b5a9813d7d8b8dc3ea37ad376ec2dbd6199a07ad843` |

Clean packages are rooted at `artifacts/tls-key-update-clean-packages-e414416`. The extracted Windows and Linux three-package smokes are rooted at `artifacts/tls-key-update-clean-extracted-smoke-win-e414416` and `artifacts/tls-key-update-clean-extracted-smoke-linux-e414416`.

| Runtime | Completed | Failed | Timed out | KeyUpdates requested | Target observations | Post-update bytes each direction | Total transferred bytes |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Windows x64 | 1 | 0 | 0 | 1 | 1 | 65,536 | 131,072 |
| Linux x64 | 1 | 0 | 0 | 1 | 1 | 65,536 | 131,072 |

Both smokes proved request byte zero (`update_not_requested`), client-write and server-read generation `0 -> 1`, unchanged reverse-direction generation, zero published traffic secrets, exact payload hash `944044fe482bc4e91085c15c5a923a1b9e02eac98d3bce04997d6dbecd2a5b8d`, exact TLS/profile/certificate/session facts, and Protocol Execution Result v2 schema conformance. The Windows smoke also proved explicit unsupported evidence for `tls.handshake.full`, `tls.handshake.resumed`, `tls.handshake.full.tls12`, `tls.handshake.full.chacha20`, `tls.handshake.mutual-auth`, `tls.early-data.accepted`, `tls.early-data.rejected`, `tls.record.coverage`, and `tls.record.throughput`; unknown input returned the distinct configuration exit.

Verification commands:

```powershell
pwsh ./scripts/package/Test-OpenSslTls13KeyUpdateSource.ps1
pwsh ./scripts/package/Validate-ProtocolLabComponentManifests.ps1
pwsh ./scripts/package/Test-ProtocolLabPackageBuildAttestation.ps1 -PackagePath <package> -AttestationPath <attestation> -RequireParityEligible
pwsh ./scripts/package/Test-OpenSslTls13KeyUpdateThreePackageSmoke.ps1 -RuntimeIdentifier win-x64 -PackageRoot ./artifacts/tls-key-update-clean-packages-e414416 -ArtifactRoot ./artifacts/tls-key-update-clean-extracted-smoke-win-e414416
pwsh ./scripts/package/Test-OpenSslTls13KeyUpdateThreePackageSmoke.ps1 -RuntimeIdentifier linux-x64 -PackageRoot ./artifacts/tls-key-update-clean-packages-e414416 -ArtifactRoot ./artifacts/tls-key-update-clean-extracted-smoke-linux-e414416
go -C implementations/go-tls13/source test -race -count=1 ./...
go -C implementations/go-tls13/source vet ./...
go -C executors/go-tls13-executor/source test -race -count=1 ./...
go -C executors/go-tls13-executor/source vet ./...
```

This delivery is local diagnostic component evidence only. It makes no runner-admission, benchmark, comparison, ranking, publication, deployment, or lab-infrastructure claim.
