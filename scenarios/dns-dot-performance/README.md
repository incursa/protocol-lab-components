# DNS-over-TLS performance scenario pack

Package version `0.2.0` adds the authoritative `dns.dot.interoperability.query.a` scenario and `dns-dot-interoperability-smoke` suite while preserving the strict v1 scenario and suite.

All authority material, including both TLS profiles, is byte-for-byte hash locked to public ProtocolLab commit `c0475b05cb80362760ac57e58ecfa1610a766c10`.

It contains declarative authority material only. Executor behavior, target behavior, implementation support, and benchmark evidence are owned by their separate packages and the runner. The smoke suite is diagnostic and non-publishable.
