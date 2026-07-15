# BIND 9 DoH2 Authority

Package version `0.1.0` runs unmodified BIND 9.20.24 as a local authoritative DoH server over exact HTTP/2. It supports `dns.doh2.interoperability.query.a` with `POST /dns-query`, the exact `plab.test` fixture, TLS, and ALPN `h2`. No proxy, HTTP fallback, recursion, cache, or external upstream is present.

Strict `dns.doh2.query.a` remains explicitly unsupported because it requires response `Cache-Control: no-store`, which unmodified BIND does not emit. The RFC 8484 interoperability lane permits absent response cache metadata.

- Source version: `9.20.24`
- Source commit: `e5d43f1764259da929f6c0200a9db2081141998a`
- Image digest: `sha256:4e2ce9999405e69bfbd1557bfa0208df88f22c444a136a0ec81fb7748be97782`
- License: `MPL-2.0`
- Executor: `go-dns-doh2-executor`

Certificate and private-key files are deterministic test-only fixtures and must not enter evidence.
