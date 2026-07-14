# TLS endpoint/tool wrapper feasibility

Updated: 2026-07-14

This audit applies the implementation-diversity rule that an unmodified
upstream executable must expose every control needed by a declared ProtocolLab
row. A library capability is not enough when its shipped example or utility
does not expose that capability. All four candidates remain in the
`tls-endpoint-tool` diagnostic cohort and are ineligible for production-origin
ranking.

## Implemented wrappers

| Candidate | Pinned upstream | Decision | Exact rows |
| --- | --- | --- | --- |
| OpenSSL `s_server` | OpenSSL `3.3.0`, tag object `24e7fcf7aff2caadbdee879f615c63981ed132dc`, source commit `4cb31128b5790819dfeea2739fbde265f71a10a2`, Apache-2.0 | `0.1.1` Docker target admitted with source archive and base-image digests; no host OpenSSL dependency | `tls.handshake.full` only |
| GnuTLS `gnutls-serv` | GnuTLS `3.8.9`, tag object `011bda1be01e4a47224adb3cbc32fcb06cba7be1`, source commit `477a733247460b94cd2b37a10579c27ca6fc196f`, GPL-3.0-or-later | `0.1.1` Docker target admitted with source archive and base-image digests; no host GnuTLS dependency | `tls.handshake.full` only |

The upstream CLIs expose enough controls for the full TLS 1.3 row: certificate
and key, TLS version, AES-128-GCM-SHA256, X25519, ECDSA P-256/SHA-256, and ALPN.
Tickets are disabled to keep the row fresh. The existing Go executor validates
the actual negotiation and canonical certificate identity.

Version `0.1.1` removes the prior exact host-tool capability gates. Each
package now source-builds its upstream release inside a Docker target pinned by
release archive SHA-256 and Debian base-image digest. This makes the targets
self-contained on Linux x64 ProtocolLab workers that provide Docker.

Neither tool implements the `PLABTLS1` record command protocol, so echo,
reverse, HTTP, record-fragment, or buffer-size modes cannot stand in for
`tls.record.throughput` or `tls.record.coverage`. Both can require and validate
a client chain, but neither CLI can pin the canonical client leaf DER and SPKI,
so `tls.handshake.mutual-auth` is not declared. Their ticket and early-data
controls also do not supply the exact existing executor lifecycle and
application-effect evidence. Every excluded row and reason is retained in each
entry manifest.

Repository evidence:

- OpenSSL `apps/s_server.c` and the matching 3.3 documentation expose the
  `-tls1_3`, `-ciphersuites`, `-groups`, `-sigalgs`, `-alpn`, `-Verify`, ticket,
  early-data, fragment, echo/reverse, and HTTP switches, but no canonical-leaf
  client-certificate pin or ProtocolLab record responder:
  https://github.com/openssl/openssl/blob/openssl-3.3.0/apps/s_server.c
- GnuTLS `src/serv.c` exposes priority strings, ALPN, SNI, certificates,
  client-certificate CA verification, tickets, early data, echo, HTTP, and MTU,
  but no canonical-leaf pin or ProtocolLab record responder:
  https://gitlab.com/gnutls/gnutls/-/blob/3.8.9/src/serv.c

## rustls upstream example server: closed, not packaged

Audit snapshot: rustls commit
`b597a77ebca764bc0950a93a56149b62a2e9b73f` (2026-07-14), dual
Apache-2.0/ISC/MIT repository licensing.

The repository itself labels `simpleserver.rs` as minimal and
`tlsserver-mio.rs` as its configurable example. The minimal server accepts only
certificate and key paths, binds a hard-coded port, uses the default provider,
disables client authentication, sends `Hello from the server` immediately, and
accepts one connection. It therefore violates the zero-application-byte full
handshake row and exposes none of the required version, suite, group, ALPN,
ticket, or authentication controls.

`tlsserver-mio.rs` is closer: it exposes TLS versions, cipher suites, ALPN,
certificate/key, CA-based client authentication, session storage/tickets, and
early-data size. It does not expose a key-exchange group selector or an exact
canonical-client-leaf verifier. Its echo/HTTP/forward modes do not implement
the `PLABTLS1` record responder. Consequently no unmodified upstream rustls
example exposes every control for any current row under the wrapper-only rule.

Repository evidence at the audited commit:

- example inventory and intended roles:
  https://github.com/rustls/rustls/blob/b597a77ebca764bc0950a93a56149b62a2e9b73f/examples/README.md
- minimal server behavior:
  https://github.com/rustls/rustls/blob/b597a77ebca764bc0950a93a56149b62a2e9b73f/examples/src/bin/simpleserver.rs
- configurable server arguments and echo/HTTP/forward modes:
  https://github.com/rustls/rustls/blob/b597a77ebca764bc0950a93a56149b62a2e9b73f/examples/src/bin/tlsserver-mio.rs
- license evidence:
  https://github.com/rustls/rustls/tree/b597a77ebca764bc0950a93a56149b62a2e9b73f#license

Follow-up trigger: reconsider only if an upstream executable adds an explicit
group selector, exact client-leaf verification when mTLS is claimed, and an
application mode matching a current contract (or a future handshake-only
contract is explicitly designed around the shipped lifecycle).

## s2n-tls `s2nd`: closed, not packaged

Audit snapshot: s2n-tls commit
`e12e7a377b69f58bc370440e5c952809b038862c` (2026-07-10), Apache-2.0.

The upstream `s2nd` utility exposes certificate/key, named cipher-preference
policy, ALPN, CA-based mutual authentication, session-ticket key, ticket
disablement, early-data size, record-size preferences, HTTPS, echo, and a
negotiate-only mode. Its `--tls13` option is explicitly a deprecated no-op in
the audited source. It exposes no command-line key-exchange group selector and
no canonical-client-leaf identity pin; a named cipher policy is not an exact
suite-and-group control. Echo and HTTPS modes do not implement `PLABTLS1`.
Therefore the unmodified server utility cannot make an exact current support
claim and is not packaged.

Repository evidence at the audited commit:

- `s2nd` usage, option table, deprecated `--tls13`, configuration, and
  echo/HTTPS/negotiate behavior:
  https://github.com/aws/s2n-tls/blob/e12e7a377b69f58bc370440e5c952809b038862c/bin/s2nd.c
- license evidence:
  https://github.com/aws/s2n-tls/blob/e12e7a377b69f58bc370440e5c952809b038862c/LICENSE

Follow-up trigger: reconsider only if the shipped utility exposes exact TLS
version, cipher suite, and group selection plus the row-specific peer identity
or application semantics. Building a new s2n server around the library would
be a different thin-adapter proposal, not this wrapper candidate.
