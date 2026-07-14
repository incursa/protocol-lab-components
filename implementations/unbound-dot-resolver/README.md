# Unbound DoT resolver feasibility decision

Working identity: org.protocol-lab.components.implementation.unbound-dot-resolver.

Closed by prerequisite on the reconciled contract baseline. The only DoT scenario, dns.dot.query.a, requires the authoritative-server role, authoritative-answer behavior, recursion unavailable, cache disabled, and an origin-server execution profile. Unbound is a recursive validating caching resolver. Registering it against that scenario would be a false role and comparison claim, while package-v2 requires every provided implementation to name at least one scenario.

Do not publish or admit an Unbound DoT package until a resolver-specific DoT scenario, suite, comparison group, and ranking policy are committed. Upstream feasibility is otherwise positive: Unbound 1.22.0 has TLS service support; source tag commit 0076736fc40298eb6252705e6e158462c6b24d06, BSD-3-Clause.

