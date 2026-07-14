# DNS-over-TLS performance scenario pack

Package version `0.1.1` declares the bundled `dns-dot-performance-smoke` suite without changing public authority bytes.

This package reproduces and hash-locks the exact `dns.dot.query.a` scenario, `dns-dot-performance-smoke` suite, `secure-dns-smoke` load profile, canonical A-query wire fixture, and secure-DNS TLS profile from public ProtocolLab authority commit `8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574`.

It contains declarative authority material only. Executor behavior, target behavior, implementation support, and benchmark evidence are owned by their separate packages and the runner. The smoke suite is diagnostic and non-publishable.
