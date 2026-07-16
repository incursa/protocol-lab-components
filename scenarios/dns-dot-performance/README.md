# DNS-over-TLS performance scenario pack

Package version `0.3.0` adds the cold-cache recursive-resolver `dns.dot.resolver.interoperability.query.a` scenario and `dns-dot-resolver-interoperability-smoke` suite while preserving the strict and authoritative lanes.

All authority material, including the resolver fixture and both TLS profiles, is byte-for-byte hash locked to its public ProtocolLab source commit.

It contains declarative authority material only. Executor behavior, target behavior, implementation support, and benchmark evidence are owned by their separate packages and the runner. The smoke suite is diagnostic and non-publishable.
