# DNS-over-HTTPS HTTP/2 performance scenario package

Package version `0.2.0` adds the authoritative `dns.doh2.interoperability.query.a` scenario and `dns-doh2-interoperability-smoke` suite while preserving the strict v1 scenario and suite.

All authority material, including both TLS profiles, is byte-for-byte hash locked to public ProtocolLab commit `c0475b05cb80362760ac57e58ecfa1610a766c10`. Strict v1 continues to require `Cache-Control: no-store`; the interoperability scenario follows RFC 8484 and accepts a response with no explicit cache metadata.

It is declarative authority material only. It does not claim executor, target, benchmark, ranking, or publishability support.
