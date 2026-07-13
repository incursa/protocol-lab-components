# TLS 1.3 KeyUpdate component delivery

Authority: `protocol-lab@8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574`.

This component-only slice adds `org.protocol-lab.components.executor.openssl-tls13-key-update-executor@0.1.0` and independent target `org.protocol-lab.components.implementation.openssl-tls13-key-update@0.1.0`. It reuses the authority-locked `org.protocol-lab.components.scenario.tls13-handshake-performance@0.2.0` package, whose lock covers `tls.key-update.diagnostic`, `tls-diagnostic`, and `plab-tls13-aes128gcm-p256-server-auth-v2`.

The only supported identity is `tls.key-update.diagnostic`. The executor and target fail closed for every other committed TLS identity as explicit `unsupported`; unknown identities remain distinct configuration errors.

The client executor calls OpenSSL's public `SSL_key_update(SSL_KEY_UPDATE_NOT_REQUESTED)` API exactly once. Its message callback proves one outgoing TLS 1.3 KeyUpdate carrying request byte zero. The independent target's message callback proves one incoming KeyUpdate and zero outgoing updates. Successful decryption of the exact 65,536-byte repeated-`0x5A` post-update payload proves the client-write/server-read traffic generation transition; the deterministic 65,536-byte response proves bidirectional connection continuity. The proof records generation numbers and event counts but never enables key logging or publishes traffic-secret material.

Windows packages use package-local OpenSSL `3.3.0` libraries. Linux packages statically link OpenSSL `3.5.7` using the pinned Alpine `3.22.2` image digest. Both packages include Apache-2.0 license material.

## Clean package evidence

Clean package hashes and extracted-package evidence will be recorded in a follow-up evidence-only commit after this implementation commit establishes a clean source head.

Verification commands:

```powershell
pwsh ./scripts/package/Test-OpenSslTls13KeyUpdateSource.ps1
pwsh ./scripts/package/Validate-ProtocolLabComponentManifests.ps1
pwsh ./scripts/package/Test-OpenSslTls13KeyUpdateThreePackageSmoke.ps1 -RuntimeIdentifier win-x64
pwsh ./scripts/package/Test-OpenSslTls13KeyUpdateThreePackageSmoke.ps1 -RuntimeIdentifier linux-x64
```

This delivery is local diagnostic component evidence only. It makes no runner-admission, benchmark, comparison, ranking, publication, deployment, or lab-infrastructure claim.
