# DNS-over-HTTPS HTTP/3 performance scenario package

Package version `0.2.0` authority-locks the seven strict DoH3 scenarios plus `dns.doh3.interoperability.query.a`, four suites, two referenced load profiles, deterministic DNS fixtures, and both secure-DNS TLS profiles at public authority commit `c0475b05cb80362760ac57e58ecfa1610a766c10`.

Every `providedSuites` entry declares `protocols: [doh3]`; this fixes the invalid 0.1.0 generated suite metadata without changing strict scenario bytes.

`secure-dns-smoke` is executable locally. The packaged comparison candidate remains non-publishable contract material and is not a benchmark claim.
