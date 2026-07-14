# BIND 9 DoT Authority

Authoritative local plab.test fixture serving 192.0.2.1 with TTL zero; recursion and external upstreams are disabled.

BIND 9 is MPL-2.0; this package ships only ProtocolLab configuration and an immutable upstream image reference.

- Version: `9.20.24`
- Source commit: `e5d43f1764259da929f6c0200a9db2081141998a`
- Image digest: `sha256:4e2ce9999405e69bfbd1557bfa0208df88f22c444a136a0ec81fb7748be97782`
- License: `MPL-2.0`
- Scenario: `dns.dot.query.a`
- Executor: `go-dns-dot-executor`

Certificate/private-key files are deterministic test-only fixtures and must not enter benchmark evidence.

