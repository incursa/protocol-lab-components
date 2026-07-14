# BIND 9 DoH2 feasibility decision

Working identity: org.protocol-lab.components.implementation.bind9-doh2.

Closed by exact-contract mismatch. BIND 9.20.24 from immutable ISC image digest sha256:4e2ce9999405e69bfbd1557bfa0208df88f22c444a136a0ec81fb7748be97782 (source commit e5d43f1764259da929f6c0200a9db2081141998a) was built with libnghttp2, its TLS/HTTP configuration passed named-checkconf, and the exact HTTP/2 executor reached the target. Validation failed at the response header gate: unmodified BIND does not provide the exact Cache-Control: no-store header required by dns.doh2.query.a. Adding a reverse proxy or modifying BIND would violate the wrapper-only scope and change the implementation under test.

No package is registered. BIND 9 is MPL-2.0.

