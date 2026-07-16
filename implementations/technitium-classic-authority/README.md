# Technitium Classic DNS Authority

Package `org.protocol-lab.components.implementation.technitium-classic-authority@0.1.0` runs the unmodified Technitium DNS Server 15.4 authoritative engine over classic DNS UDP. The source/config-only package pins the official Linux amd64 image digest, denies recursion, creates a local `plab.test` primary zone through Technitium's HTTP API, and installs the deterministic TTL-zero `A 192.0.2.1` answer.

The `.plabpkg` contains the Dockerfile and bootstrap configuration only. It does not redistribute Technitium binaries or image layers; a compatible worker acquires the GPL-3.0-only upstream image by immutable digest.

Only `dns.classic.udp.query.a` is declared. UDP passes the exact validity gate and the complete five-second diagnostic load with no malformed, retried, failed, or timed-out operations. Technitium's TCP endpoint passes a one-query validity check but closes the workload's reused connection after roughly 2,100 measured queries; the executor records `EOF`, which violates the zero-failure and connection-reuse contract, so `dns.classic.tcp.query.a` remains explicit unsupported. The synthetic byte-exact truncation fixture is also unsupported because an ordinary standards-compliant truncated answer is not the contract fixture. DoT, DoH2, DoH3, and DoQ remain separate closed feasibility decisions under the current encrypted-transport contracts.

Build with `pwsh scripts/package/Build-TechnitiumClassicAuthorityPackage.ps1`. Run `pwsh implementations/technitium-classic-authority/run.ps1 -ProofOnly` to confirm the pinned runtime version, or omit `-ProofOnly` to serve UDP on port `15355`.
