# Technitium Classic DNS Authority

Package `org.protocol-lab.components.implementation.technitium-classic-authority@0.1.1` runs the unmodified Technitium DNS Server 15.4 authoritative engine over classic DNS UDP and TCP. The source/config-only package pins the official Linux amd64 image digest, denies recursion, removes Technitium's production per-client QPM limits inside the isolated fixture, creates a local `plab.test` primary zone through Technitium's HTTP API, and installs the deterministic TTL-zero `A 192.0.2.1` answer.

The `.plabpkg` contains the Dockerfile and bootstrap configuration only. It does not redistribute Technitium binaries or image layers; a compatible worker acquires the GPL-3.0-only upstream image by immutable digest.

Only `dns.classic.udp.query.a` and `dns.classic.tcp.query.a` are declared. Package `0.1.0` exposed an exact live failure: the upstream defaults cap an IPv4 `/32` at 600 queries per minute, causing the unpaced calibration workload to drop UDP requests and close the reused TCP connection. Version `0.1.1` removes the IPv4 and IPv6 QPM prefix limits through the supported settings API before declaring readiness. That configuration is part of the fixture and does not describe Technitium's production defaults. The synthetic byte-exact truncation fixture remains unsupported because an ordinary standards-compliant truncated answer is not the contract fixture. DoT, DoH2, DoH3, and DoQ remain separate closed feasibility decisions under the current encrypted-transport contracts.

Build with `pwsh scripts/package/Build-TechnitiumClassicAuthorityPackage.ps1`. Run `pwsh implementations/technitium-classic-authority/run.ps1 -ProofOnly` to confirm the pinned runtime version, or omit `-ProofOnly` to serve UDP and TCP on port `15355`.
