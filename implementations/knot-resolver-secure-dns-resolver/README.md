# Knot Resolver secure DNS recursive resolver

Package version `0.1.0` runs digest-pinned Knot Resolver 6.4.0 as the measured native DNS-over-TLS and DNS-over-HTTPS HTTP/2 recursive resolver. Knot Resolver's own listeners terminate TLS and HTTP/2; no reverse proxy or protocol adapter substitutes for the resolver.

The package forwards only `plab.test.` to a loopback-only deterministic UDP authority, flushes the resolver cache through `kresctl` before every operation, and exposes only a narrow HTTP control bridge to the executor. DNSSEC validation remains enabled generally and the isolated unsigned fixture zone is explicitly marked insecure. Resolver evidence is diagnostic, non-publishable, and never compared with authoritative-server rows.
