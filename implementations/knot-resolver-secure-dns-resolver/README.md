# Knot Resolver secure DNS recursive resolver

Package version `0.1.4` runs digest-pinned Knot Resolver 6.4.0 as the measured native DNS-over-TLS and DNS-over-HTTPS HTTP/2 recursive resolver. Knot Resolver's own listeners terminate TLS and HTTP/2; no reverse proxy or protocol adapter substitutes for the resolver.

The package forwards the root subtree to a loopback-only deterministic UDP authority on the standard DNS port so background priming and measured queries cannot use an external upstream. The fixture answers only `plab.test.`, the package flushes the resolver cache through `kresctl` before every operation, and it exposes the same narrow HTTP control bridge on each executor-default control port after Knot's management API is ready. DNSSEC validation remains enabled generally and the isolated unsigned fixture zone is explicitly marked insecure. Resolver evidence is diagnostic, non-publishable, and never compared with authoritative-server rows.
