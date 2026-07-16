# Unbound DoH2 recursive resolver

Package version `0.1.1` runs digest-pinned Unbound 1.22.0 as the measured native DNS-over-HTTPS HTTP/2 recursive resolver. The image's own libnghttp2-backed listener terminates TLS and HTTP/2; no reverse proxy or protocol adapter substitutes for Unbound.

The package forwards only `plab.test.` to a loopback-only deterministic UDP authority, rejects other names, flushes the resolver cache through Unbound's local control protocol before every operation, and exposes only a narrow HTTP control bridge to the executor. DNSSEC validation remains enabled generally and the isolated unsigned fixture zone is explicitly marked insecure. Resolver evidence is diagnostic, non-publishable, and never compared with authoritative-server rows.
