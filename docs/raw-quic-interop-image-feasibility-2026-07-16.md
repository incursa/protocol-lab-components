# Raw QUIC Interop Image Feasibility Decisions

Date: 2026-07-16

This record evaluates compatibility-first raw QUIC candidates whose upstream
interop image might expose ProtocolLab's exact application stream contract. A
successful HTTP/3 or HQ peer does not count as a raw QUIC target. These
decisions do not reduce the raw QUIC coverage floor, which is already met by
independent live implementations.

## ngtcp2

Candidate image: `ghcr.io/ngtcp2/ngtcp2-interop@sha256:f3703cc822d79f246bb44bbf89b6632438730c52b5c23aaa305c8bbda29f27af`

Decision: the pinned upstream image does not expose an exact raw scenario
mapping and is not packaged as `ngtcp2-raw`.

Evidence:

- The image contains `wsslserver`/`wsslclient` for HTTP/3 and
  `wsslhqserver`/`wsslhqclient` for the HQ interop protocol. The complete
  `wsslhqserver --help` surface has no application-protocol or ALPN override.
- ngtcp2's own documentation describes the HQ binaries as specifically
  tailored for `quic-interop-runner`; it does not describe them as arbitrary
  raw stream servers.
- A local isolated Docker-network probe started the pinned `wsslhqserver` and
  invoked the package-backed quic-go raw executor with ALPN `plab-raw-quic`,
  `handshake-cold`, one connection, zero streams, and fail-on-errors enabled.
  The exact result was one attempted connection, zero successful connections,
  error rate 1, client error
  `CRYPTO_ERROR 0x178 (remote): tls: no application protocol`, and server log
  `ngtcp2_conn_read_pkt: ERR_CRYPTO`.

The existing `ngtcp2-http3` package remains valid HTTP/3 peer
characterization. Relabeling either the HTTP/3 or HQ binary as raw QUIC would
change the application protocol and violate the wishlist's no-substitution
rule.

## Re-entry condition

Re-open ngtcp2 raw packaging only if an immutable upstream binary exposes a
custom ALPN and byte-for-byte stream handler, or if ProtocolLab explicitly
approves and owns a separate native ngtcp2 server application with its TLS,
UDP event loop, callbacks, flow control, build toolchain, and lifecycle fully
maintained. That would be a new adapter implementation, not a wrapper around
the current interop image.

## Sources

- ngtcp2 repository and interop-binary descriptions: <https://github.com/ngtcp2/ngtcp2>
- ngtcp2 programmers' guide: <https://nghttp2.org/ngtcp2/programmers-guide.html>
- Existing pinned-image record: `implementations/ngtcp2-http3/protocol-lab.internal.json`
- Existing HTTP/3 wrapper boundary: `implementations/ngtcp2-http3/README.md`

