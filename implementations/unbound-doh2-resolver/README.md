# Unbound DoH2 resolver feasibility decision

Working identity: org.protocol-lab.components.implementation.unbound-doh2-resolver.

Closed by prerequisite on the reconciled contract baseline. The only DoH2 scenario, dns.doh2.query.a, requires the authoritative-server role, local authoritative mode, recursion unavailable, cache disabled, and an origin-server execution profile. Unbound is a recursive validating caching resolver. Registering it against that scenario would be a false role and comparison claim, while package-v2 requires every provided implementation to name at least one scenario.

Do not publish or admit an Unbound DoH2 package until a resolver-specific DoH2 scenario, suite, comparison group, and ranking policy are committed. Upstream feasibility is otherwise positive: Unbound 1.22.0 was built with libnghttp2; source tag commit 0076736fc40298eb6252705e6e158462c6b24d06, BSD-3-Clause.

