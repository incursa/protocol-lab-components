# DNS-over-HTTPS HTTP/2 performance scenario package

Package version `0.3.0` adds the recursive-resolver `dns.doh2.resolver.interoperability.query.a` scenario, deterministic resolver fixture, and resolver-specific suite while preserving the strict and authoritative lanes.

All authority material, including both TLS profiles, is byte-for-byte hash locked to public ProtocolLab commit `a5ac2dd6bdc4facd175b49747c387bdebb33ab38`. Strict v1 continues to require `Cache-Control: no-store`; both interoperability scenarios follow RFC 8484 and accept a response with no explicit cache metadata. Resolver and authoritative evidence remain separate cohorts.

It is declarative authority material only. It does not claim executor, target, benchmark, ranking, or publishability support.
