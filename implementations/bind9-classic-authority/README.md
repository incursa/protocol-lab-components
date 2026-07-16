# BIND 9 Classic DNS Authority

Package `org.protocol-lab.components.implementation.bind9-classic-authority@0.1.0` runs unmodified BIND 9.20.24 as a local authoritative server over classic DNS UDP and TCP. It serves the deterministic `plab.test.` A answer with TTL zero, disables recursion and external upstreams, and publishes the same selected port over both transports.

The package claims only `dns.classic.udp.query.a` and `dns.classic.tcp.query.a`. It explicitly does not claim the synthetic byte-exact DNSSEC-shaped truncation fixture or any encrypted DNS transport. BIND is MPL-2.0; this package ships ProtocolLab configuration and an immutable upstream image reference, not BIND binaries.

Use `run.ps1 -ProofOnly` or `PLAB_PROOF_ONLY=true ./run.sh` to prove the pinned runtime version. The real ProtocolLab controller must select the UDP and TCP executors separately; their outputs remain distinct evidence cells and are not an encrypted-versus-cleartext performance conclusion by themselves.
