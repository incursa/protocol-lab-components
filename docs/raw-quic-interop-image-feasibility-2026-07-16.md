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

## XQUIC

Candidate image:
`ghcr.io/alibaba/xquic/xquic-interop@sha256:875df1e9935c6a07e26d7b5ae14df9edd06703061ce35920234a97d6991c58e0`

Decision: the pinned upstream image does not expose an exact raw scenario
mapping and is not packaged as `xquic-raw`.

Evidence:

- `run_endpoint.sh` accepts only named QUIC-interoperability testcases and
  launches `/xquic_bin/demo_server` in interop/file-server mode.
- The complete `demo_server` option surface has no ALPN or arbitrary
  application-handler control.
- A canonical `plab-raw-quic` cold-handshake probe against the running server
  attempted one connection and completed zero. The executor reported
  `FRAME_ENCODING_ERROR (local) (frame type: 0x1d): 29 not allowed at
  encryption level Initial` and error rate 1.

XQUIC remains a valid separate HTTP/3 packaging candidate. Its interop binary
cannot be relabeled as the ProtocolLab raw application.

## LSQUIC

Candidate image:
`litespeedtech/lsquic-qir@sha256:7819190d3b19ac4f1a9b85d994fdc3a6de900973376d1c7dad345641c3fb28c8`

Decision: the pinned upstream image does not expose an exact raw scenario
mapping and is not packaged as `lsquic-raw`.

Evidence:

- `run_endpoint.sh` always launches `/usr/bin/http_server` as an HTTP/3 or HQ
  file server. Its non-HTTP/3 interop rows select `hq-interop`.
- `http_server -Q ALPN` can assign an arbitrary ALPN, but `-Q` explicitly
  selects HQ mode; it does not replace the file-server application callbacks
  with a byte-for-byte stream handler.
- A canonical 1 KiB echo probe deliberately launched the server with
  `-Q plab-raw-quic`. QUIC negotiation completed and the executor wrote exactly
  1,024 bytes, but it received zero response bytes. The exact failure was
  `received 0 response bytes, expected 1024`, with one attempted request, zero
  successful requests, and error rate 1.

LSQUIC remains a valid separate HTTP/3 packaging candidate. Successful custom
ALPN negotiation alone is not evidence for the raw application contract.

## neqo

Candidate image:
`ghcr.io/mozilla/neqo-qns@sha256:0d97da7602b40315768d0780de549b8b15d2d9cfd0e26ac948b0d6809c7f1571`

Decision: the pinned upstream image does not expose an exact raw scenario
mapping and is not packaged as `neqo-raw`.

Evidence:

- `/neqo/interop.sh` launches `neqo-server --qns-test` for the upstream
  network-simulator testcases.
- `neqo-server --help` exposes `--alpn`, but explicitly states that the binary
  still only executes HTTP/3 regardless of the label.
- A canonical 1 KiB echo probe launched the server with
  `--alpn plab-raw-quic`. The executor wrote exactly 1,024 bytes, received zero,
  and reported `stream 0 canceled by remote with error code 0`, with one
  attempted request, zero successful requests, and error rate 1.

neqo remains a valid separate HTTP/3 packaging candidate. Its test server is
not a generic raw-stream server.

## mvfst / Proxygen

Candidate image:
`ghcr.io/facebook/proxygen/mvfst-interop@sha256:36b395ccdcd94339120572c0ff4d1f2eafe76eeeb01a1ac8c405182625303819`

Decision: the pinned upstream image does not expose an exact raw scenario
mapping and is not packaged as `mvfst-raw`.

Evidence:

- `run_endpoint.sh` launches Proxygen's `hq` sample, using `hq-interop` and
  HTTP/0.9 for transport interop rows or `h3` and HTTP/1.1 for the HTTP/3 row.
- The binary's command surface is HTTP transaction oriented: protocol and HTTP
  version selection, request paths, a static document root, and Proxygen
  sample handlers. It does not expose a generic byte-stream callback.
- A canonical 1 KiB echo probe launched the server with
  `--protocol=plab-raw-quic`. The server recorded
  `next protocol not supported: plab-raw-quic` and `ALPN not supported`; the
  executor completed zero requests and received a remote `INTERNAL_ERROR`.

mvfst remains eligible for a separate transport adapter, and Proxygen remains
eligible for the wishlist's HTTP/3-origin evaluation. Neither is satisfied by
relabeling the upstream HQ sample.

### HTTP/3 origin-contract evaluation (2026-07-17)

The same pinned image was evaluated as a standalone `hq` H3 static origin,
using `--protocol=h3` and `--httpversion=1.1` on its client and the sample
server's `--static_root` option. The H3 client completed a request, and local
HTTP/1.1 checks received `200` for both `/status` and `/bytes/1024`. The
responses carried no `Content-Type` header. Consequently the sample cannot
satisfy the current ProtocolLab origin contract, whose status and byte rows
require typed JSON and `application/octet-stream` responses. It is not
packaged as an HTTP/3 origin and is not evidence for the origin cohort.

Re-open the HTTP/3-origin task only when an immutable upstream Proxygen
surface emits the required typed responses, or when ProtocolLab explicitly
approves ownership of a separately maintained native Proxygen application
adapter. Adding headers outside the H3 handler would not make the upstream
sample a contract-compliant origin.

## Re-entry conditions

Re-open ngtcp2 raw packaging only if an immutable upstream binary exposes a
custom ALPN and byte-for-byte stream handler, or if ProtocolLab explicitly
approves and owns a separate native ngtcp2 server application with its TLS,
UDP event loop, callbacks, flow control, build toolchain, and lifecycle fully
maintained. That would be a new adapter implementation, not a wrapper around
the current interop image.

Apply the same boundary to XQUIC, LSQUIC, neqo, and mvfst: re-open a raw package
only when a pinned upstream executable exposes both the exact ALPN and exact
byte-stream behavior, or when ProtocolLab owns and maintains a native adapter
whose application callbacks implement the canonical raw wire contract.

These raw decisions do not close the later wishlist items for honest HTTP/3
packages. XQUIC, LSQUIC, and neqo still require reproducibly pinned HTTP/3
packages and live evidence; Proxygen still requires an origin-contract
evaluation.

## Sources

- ngtcp2 repository and interop-binary descriptions: <https://github.com/ngtcp2/ngtcp2>
- ngtcp2 programmers' guide: <https://nghttp2.org/ngtcp2/programmers-guide.html>
- XQUIC repository: <https://github.com/alibaba/xquic>
- LSQUIC repository: <https://github.com/litespeedtech/lsquic>
- neqo repository: <https://github.com/mozilla/neqo>
- mvfst repository: <https://github.com/facebook/mvfst>
- Proxygen repository: <https://github.com/facebook/proxygen>
- Existing pinned-image record: `implementations/ngtcp2-http3/protocol-lab.internal.json`
- Existing HTTP/3 wrapper boundary: `implementations/ngtcp2-http3/README.md`
