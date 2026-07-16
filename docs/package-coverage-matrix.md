# ProtocolLab Package Coverage Matrix

Observed: 2026-07-15

This matrix tracks reusable component package coverage for implementations and executors that should be visible in ProtocolLab comparisons. It separates package inventory from proof quality: `supported` means the package metadata and wrapper advertise the row; live benchmark or conformance evidence remains tied to report artifacts.

## Inputs

| Source | Observed entries |
| --- | --- |
| Component packages before this change | `aioquic-http3`, `quiche-http3`, `ngtcp2-http3`, `kestrel-http3`, `quic-go-http3`, `quic-go-raw-load`, `h3spec-http3-qpack`, `aioquic-rfc9220-websocket`, plus HTTP/1 packages and the raw QUIC scenario pack |
| Scenario packs added for controller selection | `h3spec-http3-qpack`, `http3-peer-characterization`, and `aioquic-rfc9220-websocket` focused scenario packs |
| Public site implementation catalog | `kestrel-http3`, `incursa-http3`, `msquic-dotnet`, `caddy-http3`, `nginx-http3`, and planned `quic-go-http3` entry |
| Live controller inventory after Caddy/nginx correction | `org.protocol-lab.components.implementation.caddy-http3` and `org.protocol-lab.components.implementation.nginx-http3` versions through final `0.1.7` are installed/selectable |
| Live controller final aioquic proof | `job-0d08b2ace1704d609ec9803e6e7119c7`; current `0.3.2` H3 status and 1KB validation passed, benchmarks succeeded, package-backed provenance recorded |
| Live controller final aioquic h3spec/QPACK proof | `job-a3c8b35637e14c49b86332a928c5b15d`; current `0.3.3` h3spec status, response-header, and QPACK diagnostics passed with exact executor identity and requested/effective load shapes retained |
| Live controller final quic-go proof | `job-610e9f2d38364cfc95b238ea6e012446`; current `0.1.6` H3 status, 1KB, and 64KB validation passed, benchmarks succeeded, package-backed provenance recorded |
| Live controller final Kestrel proof | `job-fb08e6a527b94ee1a922055a9401feee`; current `0.1.6` H3 status, 1KB, and 64KB validation passed, benchmarks succeeded, package-backed provenance recorded |
| Live controller final quic-go raw proof | `job-43983c8c7a35400fa54767ba0be66045`; raw stream throughput and multiplex validation passed, benchmark succeeded, package-backed provenance recorded |
| Live controller final Caddy proof | `job-c92b1918b59846018dcc808babd58730`; H3 status, 1KB, and 64KB validation passed, benchmark succeeded, package-backed provenance recorded |
| Live controller final Caddy/nginx correction proof | `job-e30007af97354306ab7d34abbbbbab4a`; Caddy and nginx H3 status, 1KB, and 64KB validation passed, benchmark succeeded with final `0.1.7` packages; controller-host publication lacked `CLOUDFLARE_API_TOKEN`, so the staged bundle was uploaded with local Wrangler auth, queued, imported into D1, and appeared on live 1KB/64KB leaderboards |
| Live controller peer characterization attempt | `job-8f658e39aa0446a0a05240d580daed64`; preview selected two runnable diagnostic cells for quiche `0.1.5` and ngtcp2 `0.1.4`, but the job was cancelled after remaining in `running-benchmark` on quiche cell 1 with zero completed cells |
| Local source-context implementations | `quic-dotnet-dev`, `quic-dotnet-raw-dev`, `msquic-dotnet-raw-adapter-v1`, `incursa-http3` remain implementation-owned outside this repository |

## Before And After

| Visible implementation or executor | Before package state | After package state |
| --- | --- | --- |
| `kestrel-http3` | source package present | self-contained Linux x64 target package; current status, 1KB, and 64KB live proof retained |
| `caddy-http3` | visible on public site and present in internal consumer manifests, but not packaged in this repo | added and registered `org.protocol-lab.components.implementation.caddy-http3` |
| `nginx-http3` | present in internal consumer manifests, but not packaged in this repo | added `org.protocol-lab.components.implementation.nginx-http3`; live H3 status, 1KB, and 64KB smoke passed |
| `incursa-http3` / `quic-dotnet-dev` | live/package-owned outside this repo | unchanged; not duplicated in component repo |
| `msquic-dotnet` / `quic-dotnet-raw-dev` | live/package-owned outside this repo | unchanged; raw QUIC scenario and quic-go executor remain reusable component packages |
| `aioquic-http3` | source package present | current `0.3.3` package has live h3spec status, response-header, and QPACK proof; `0.3.2` retains historical canonical status and 1KB proof |
| `quiche-http3` | source package present | source package present with explicit scenario coverage metadata |
| `ngtcp2-http3` | source package present | source package present with explicit scenario coverage metadata |
| `quic-go-http3` | protocol-lab-internal placeholder and raw QUIC executor only | added `org.protocol-lab.components.implementation.quic-go-http3`; current status, 1KB, and 64KB live proof retained |
| `quic-go-raw` | no raw QUIC implementation package | added `org.protocol-lab.components.implementation.quic-go-raw` for `quic.transport.stream-throughput.1mb` and `quic.transport.multiplex.100x64kb` only |
| `quic-go-raw-load` | source executor package present | source executor package present with explicit comparison-lane coverage metadata |
| `h3spec-http3-qpack` | source executor package present | source executor package present with explicit comparison-lane coverage metadata |
| `aioquic-rfc9220-websocket` | source executor package present | corrected `0.3.0` exact six-ID RFC9220 executor plus origin-server `aioquic-http3@0.3.0`; it requires authenticated certificate proof, exact sustained profiles, executor/generator/parser identities, archive and immutable image digests, bounded raw artifacts, and fail-closed selection |
| `h3spec-http3-qpack` scenario pack | missing | added `org.protocol-lab.components.scenario.h3spec-http3-qpack` with suite `h3spec-http3-qpack-focused` bound to `h3spec-http3-qpack` |
| `http3-peer-characterization` scenario pack | missing | added `org.protocol-lab.components.scenario.http3-peer-characterization` with suite `http3-peer-characterization` for diagnostic external peer rows |
| HTTP/3 WebSocket scenario pack | missing | immutable corrected `org.protocol-lab.components.scenario.http3-websocket-performance@0.2.2` authority-locks the six RFC9220 v2 scenarios and exact `websocket-smoke`/`diagnostic` profiles |
| `go-http1-executor` | missing | local diagnostic package `org.protocol-lab.components.executor.go-http1-executor@0.3.0`; exact HTTP/1.1 validation plus pinned `oha@1.15.0` load generation |
| `go-http2-executor` | missing | local diagnostic package `org.protocol-lab.components.executor.go-http2-executor@0.3.0`; exact h2c smoke, diagnostic, and `16/128/8` balanced-round-robin comparison shapes |
| `apache-http1` | missing | added `org.protocol-lab.components.implementation.apache-http1@0.1.0`; unmodified digest-pinned Apache httpd, exact HTTP/1.1 plaintext/JSON executor proof, and deterministic non-advertised download/header fixtures |
| `apache-http2` | missing | added `org.protocol-lab.components.implementation.apache-http2@0.1.0`; unmodified digest-pinned Apache httpd/mod_http2 with exact h2c plaintext/JSON executor proof and a distinct TLS/ALPN startup variant retained as executor-unavailable/non-ranking |
| HTTP/2 RFC 8441 WebSocket three-package lane | missing | local diagnostic `org.protocol-lab.components.scenario.http2-websocket-performance@0.1.0`, `go-http2-websocket-executor@0.1.0`, and independent `kestrel-http2-websocket@0.1.0` packages; all six exact identities prove TLS 1.3/ALPN h2, Extended CONNECT, raw framing, masking, payloads, ordering, control frames, and close semantics |
| `go-tls13-executor` | missing | local diagnostic package `org.protocol-lab.components.executor.go-tls13-executor@0.3.0`; exact TLS 1.3 full/resumed handshakes plus deterministic record throughput and six-case record coverage; remaining committed TLS identities fail closed as explicit `unsupported` |
| `go-tls13-mtls-executor` | missing | local diagnostic package `org.protocol-lab.components.executor.go-tls13-mtls-executor@0.1.0`; exact TLS 1.3 mutual-authentication handshake with pinned server/client certificate identities; every other committed TLS identity fails closed as explicit `unsupported` |
| `go-utls-tls13-chacha20-executor` | missing | local diagnostic package `org.protocol-lab.components.executor.go-utls-tls13-chacha20-executor@0.1.0`; pinned uTLS v1.8.2 custom ClientHello offers only TLS 1.3 ChaCha20/X25519 with no ticket, PSK, or early data; every other committed TLS identity fails closed as explicit `unsupported` |
| `go-tls12-executor` | missing | local diagnostic package `org.protocol-lab.components.executor.go-tls12-executor@0.1.0`; exact TLS 1.2 compatibility handshake with pinned cipher, X25519, ALPN, and server certificate identity; every other committed TLS identity fails closed as explicit `unsupported` |
| `rustls-tls13-early-data-executor` | missing | local diagnostic package `org.protocol-lab.components.executor.rustls-tls13-early-data-executor@0.1.0`; exact TLS 1.3 PSK-resumed accepted and rejected 0-RTT identities, with one post-handshake retry and zero duplicate effects required for rejection; every other committed TLS identity fails closed as explicit `unsupported` |
| `openssl-tls13-key-update-executor` | missing | local diagnostic package `org.protocol-lab.components.executor.openssl-tls13-key-update-executor@0.1.0`; exact client-initiated TLS 1.3 KeyUpdate with `update_not_requested`, bilateral OpenSSL message-callback observation, deterministic post-update traffic, and no traffic-secret publication; every other committed TLS identity fails closed as explicit `unsupported` |
| `dotnet-sslstream-tls13` | missing | local diagnostic package `org.protocol-lab.components.implementation.dotnet-sslstream-tls13@0.1.0`; independent library-backed TLS 1.3 target |
| `go-tls13` | missing | local diagnostic package `org.protocol-lab.components.implementation.go-tls13@0.2.0`; independent Go `crypto/tls` target for exact TLS 1.3 full/resumed handshakes and deterministic record transfers |
| `go-tls13-mtls` | missing | local diagnostic package `org.protocol-lab.components.implementation.go-tls13-mtls@0.1.0`; independent narrow Go `crypto/tls` target requiring the canonical client trust chain and exact client leaf identity |
| `go-tls13-chacha20` | missing | local diagnostic package `org.protocol-lab.components.implementation.go-tls13-chacha20@0.1.0`; independent narrow Go `crypto/tls` target for the exact TLS 1.3 ChaCha20/X25519 full-handshake profile |
| `go-tls12` | missing | local diagnostic package `org.protocol-lab.components.implementation.go-tls12@0.1.0`; independent narrow Go `crypto/tls` target for the exact TLS 1.2 ECDHE-ECDSA AES-128-GCM compatibility profile |
| `rustls-tls13-early-data` | missing | local diagnostic package `org.protocol-lab.components.implementation.rustls-tls13-early-data@0.1.0`; independent rustls target for the exact accepted and rejected TLS 1.3 early-data identities only |
| `openssl-tls13-key-update` | missing | local diagnostic package `org.protocol-lab.components.implementation.openssl-tls13-key-update@0.1.0`; independent OpenSSL target for the exact TLS 1.3 KeyUpdate diagnostic only |
| OpenSSL `s_server` | missing | Docker source-built TLS endpoint/tool package `org.protocol-lab.components.implementation.openssl-s-server@0.1.1`; exact OpenSSL 3.3.0 target with no host-tool dependency and `tls.handshake.full` only; resumption, record, mTLS, early-data, and KeyUpdate rows remain explicit unsupported |
| GnuTLS `gnutls-serv` | missing | Docker source-built TLS endpoint/tool package `org.protocol-lab.components.implementation.gnutls-serv@0.1.1`; exact GnuTLS 3.8.9 Linux target with no host-tool dependency and `tls.handshake.full` only; resumption, record, mTLS, early-data, and KeyUpdate rows remain explicit unsupported |
| HTTP/1.1 cleartext WebSocket three-package lane | missing | local diagnostic scenario, executor, and independent origin packages at `0.1.0`; exact five-ID RFC 6455 smoke passes, while all adjacent WebSocket identities fail closed as unsupported |
| HTTP/1.1 TLS WebSocket three-package lane | missing | local diagnostic `org.protocol-lab.components.scenario.http1-websocket-tls-performance`, `go-http1-websocket-tls-executor`, and independent `go-http1-websocket-tls` packages at `0.2.0`; the original five-ID TLS 1.3 regression plus exact `plab.echo.v1` and permessage-deflate no-context-takeover diagnostics pass with authenticated certificate and `http/1.1` ALPN proof |
| `go-dns-doq` | missing | local diagnostic package `org.protocol-lab.components.implementation.go-dns-doq@0.1.0`; QUIC v1/TLS 1.3/ALPN `doq` local fixture authority for `dns.doq.query.a` only |
| `go-dns-doq-executor` | missing | local diagnostic package `org.protocol-lab.components.executor.go-dns-doq-executor@0.1.0`; exact RFC 9250 DoQ binding, one query per client-initiated bidirectional stream, and no DNS-transport fallback |
| `dns-doq-performance` scenario pack | missing | authority-locked `org.protocol-lab.components.scenario.dns-doq-performance@0.1.0` for `dns.doq.query.a`, `dns-doq-performance-smoke`, and `secure-dns-smoke` at public commit `8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574` |
| DoH3 three-package lane | missing | local diagnostic `org.protocol-lab.components.scenario.dns-doh3-performance`, `org.protocol-lab.components.executor.go-dns-doh3-executor`, and independent `org.protocol-lab.components.implementation.go-dns-doh3` packages at `0.1.0`; all seven exact committed DoH3 identities pass extracted-package smoke with QUIC v1, TLS 1.3, `h3`, HTTP/3, DNS semantic, canonical hash, and no-fallback proof |

## Scenario Support

| Package | Raw QUIC | H3 1KB | H3 64KB | H3 large body | Header-heavy / QPACK | WebSocket-over-H3 | h3spec / QPACK executor |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `kestrel-http3` | unsupported | supported, live proof | supported, live proof | supported metadata | unsupported | unsupported | compatible target, unproven |
| `caddy-http3` | unsupported | supported, live proof | supported, live proof | unsupported | unsupported | unsupported | compatible target, unproven |
| `nginx-http3` | unsupported | supported, live proof | supported, live proof | skipped pending broader fixture promotion | unsupported | unsupported | compatible target, unproven |
| `aioquic-http3` | unsupported | supported, historical 0.3.2 proof | unproven | unsupported | supported, live proof | supported via RFC9220 endpoint and executor | compatible target, live proof |
| `quiche-http3` | unsupported | validation-failed; characterization supported | validation-failed; characterization supported | validation-failed; characterization supported | client-only | unsupported | compatible target, unproven |
| `ngtcp2-http3` | unsupported | validation-failed; characterization supported | validation-failed; characterization supported | validation-failed; characterization supported | client-only | unsupported | compatible target, unproven |
| `quic-go-http3` | unsupported | supported, live proof | supported, live proof | supported metadata | unsupported | unsupported | compatible target, unproven |
| `quic-go-raw` | supported target for stream throughput and multiplex only, live proof | unsupported | unsupported | unsupported | unsupported | unsupported | unsupported |
| `quic-go-raw-load` | supported executor | unsupported | unsupported | unsupported | unsupported | unsupported | unsupported |
| `h3spec-http3-qpack` | unsupported | compatible target | compatible target | compatible target | supported, aioquic live proof | unsupported | supported, aioquic live proof |
| `aioquic-rfc9220-websocket` | unsupported | unsupported | unsupported | unsupported | unsupported | supported | unsupported |
| `quic-dotnet-dev` | out of scope for this repo | owned by `quic-dotnet` package flow | owned by `quic-dotnet` package flow | owned by `quic-dotnet` package flow | owned by `quic-dotnet` package flow | owned by `quic-dotnet` package flow | target for component executors |
| `quic-dotnet-raw-dev` | owned by `quic-dotnet` package flow | out of scope | out of scope | out of scope | out of scope | out of scope | out of scope |

## Package Paths

| Package | Path |
| --- | --- |
| `org.protocol-lab.components.implementation.caddy-http3` | `implementations/caddy-http3` |
| `org.protocol-lab.components.implementation.nginx-http3` | `implementations/nginx-http3` |
| `org.protocol-lab.components.implementation.aioquic-http3` | `implementations/aioquic-http3` |
| `org.protocol-lab.components.implementation.quiche-http3` | `implementations/quiche-http3` |
| `org.protocol-lab.components.implementation.ngtcp2-http3` | `implementations/ngtcp2-http3` |
| `org.protocol-lab.components.implementation.quic-go-http3` | `implementations/quic-go-http3` |
| `org.protocol-lab.components.implementation.quic-go-raw` | `implementations/quic-go-raw` |
| `org.protocol-lab.components.implementation.kestrel-http3` | `implementations/kestrel-http3` |
| `org.protocol-lab.components.executor.quic-go-raw-load` | `executors/quic-go-raw-load` |
| `org.protocol-lab.components.executor.h3spec-http3-qpack` | `executors/h3spec-http3-qpack` |
| `org.protocol-lab.components.executor.curl-http3-client` | `executors/curl-http3-client` |
| `org.protocol-lab.components.executor.aioquic-rfc9220-websocket` | `executors/aioquic-rfc9220-websocket` |
| `org.protocol-lab.components.executor.go-http1-executor` | `executors/go-http1-executor` |
| `org.protocol-lab.components.executor.go-http2-executor` | `executors/go-http2-executor` |
| `org.protocol-lab.components.implementation.apache-http1` | `implementations/apache-http1` |
| `org.protocol-lab.components.implementation.apache-http2` | `implementations/apache-http2` |
| `org.protocol-lab.components.executor.go-http2-websocket-executor` | `executors/go-http2-websocket-executor` |
| `org.protocol-lab.components.implementation.kestrel-http2-websocket` | `implementations/kestrel-http2-websocket` |
| `org.protocol-lab.components.scenario.http2-websocket-performance` | `scenarios/http2-websocket-performance` |
| `org.protocol-lab.components.executor.go-tls13-executor` | `executors/go-tls13-executor` |
| `org.protocol-lab.components.executor.go-tls13-mtls-executor` | `executors/go-tls13-mtls-executor` |
| `org.protocol-lab.components.executor.go-utls-tls13-chacha20-executor` | `executors/go-utls-tls13-chacha20-executor` |
| `org.protocol-lab.components.executor.go-tls12-executor` | `executors/go-tls12-executor` |
| `org.protocol-lab.components.executor.rustls-tls13-early-data-executor` | `executors/rustls-tls13-early-data-executor` |
| `org.protocol-lab.components.executor.openssl-tls13-key-update-executor` | `executors/openssl-tls13-key-update-executor` |
| `org.protocol-lab.components.implementation.dotnet-sslstream-tls13` | `implementations/dotnet-sslstream-tls13` |
| `org.protocol-lab.components.implementation.go-tls13` | `implementations/go-tls13` |
| `org.protocol-lab.components.implementation.go-tls13-mtls` | `implementations/go-tls13-mtls` |
| `org.protocol-lab.components.implementation.go-tls13-chacha20` | `implementations/go-tls13-chacha20` |
| `org.protocol-lab.components.implementation.go-tls12` | `implementations/go-tls12` |
| `org.protocol-lab.components.implementation.rustls-tls13-early-data` | `implementations/rustls-tls13-early-data` |
| `org.protocol-lab.components.implementation.openssl-tls13-key-update` | `implementations/openssl-tls13-key-update` |
| `org.protocol-lab.components.implementation.openssl-s-server` | `implementations/openssl-s-server` |
| `org.protocol-lab.components.implementation.gnutls-serv` | `implementations/gnutls-serv` |
| `org.protocol-lab.components.implementation.go-http1-websocket` | `implementations/go-http1-websocket` |
| `org.protocol-lab.components.executor.go-http1-websocket-executor` | `executors/go-http1-websocket-executor` |
| `org.protocol-lab.components.scenario.http1-websocket-cleartext-performance` | `scenarios/http1-websocket-cleartext-performance` |
| `org.protocol-lab.components.implementation.go-http1-websocket-tls` | `implementations/go-http1-websocket-tls` |
| `org.protocol-lab.components.executor.go-http1-websocket-tls-executor` | `executors/go-http1-websocket-tls-executor` |
| `org.protocol-lab.components.scenario.http1-websocket-tls-performance` | `scenarios/http1-websocket-tls-performance` |
| `org.protocol-lab.components.implementation.go-dns-doq` | `implementations/go-dns-doq` |
| `org.protocol-lab.components.executor.go-dns-doq-executor` | `executors/go-dns-doq-executor` |
| `org.protocol-lab.components.scenario.dns-doq-performance` | `scenarios/dns-doq-performance` |
| `org.protocol-lab.components.implementation.go-dns-doh3` | `implementations/go-dns-doh3` |
| `org.protocol-lab.components.executor.go-dns-doh3-executor` | `executors/go-dns-doh3-executor` |
| `org.protocol-lab.components.scenario.dns-doh3-performance` | `scenarios/dns-doh3-performance` |
| `org.protocol-lab.components.scenario.raw-quic-transport` | `scenarios/raw-quic-transport` |
| `org.protocol-lab.components.scenario.h3spec-http3-qpack` | `scenarios/h3spec-http3-qpack` |
| `org.protocol-lab.components.scenario.http3-peer-characterization` | `scenarios/http3-peer-characterization` |
| `org.protocol-lab.components.scenario.http3-websocket-performance` | `scenarios/aioquic-rfc9220-websocket` |

## Registered Package

| Package | Version | SHA-256 | Controller status |
| --- | --- | --- | --- |
| `org.protocol-lab.components.implementation.caddy-http3` | `0.1.7` | `96b3c8d1859ef927dcaee0866d4b5b73e35f5a4d61a897a2d35dc8d59f28fa2d` | admitted, installed, selectable; live package-backed H3 status, 1KB, and 64KB smoke passed in `job-e30007af97354306ab7d34abbbbbab4a`; 1MB remains unsupported until package-backed live proof covers it |
| `org.protocol-lab.components.implementation.nginx-http3` | `0.1.7` | `a1bc60a99a5a8c96684d9ede91b93a0b393f6174b89d9bc8d103759e8d6a9175` | admitted, installed, selectable; live package-backed H3 status, 1KB, and 64KB smoke passed in `job-e30007af97354306ab7d34abbbbbab4a`; 1MB remains skipped/not declared until package-backed live proof covers it |
| `org.protocol-lab.components.implementation.quic-go-http3` | `0.1.6` | `faef7cb7416899a2302efaad6760ec862242a9e3eff212706fade88c8c2b14ab` | admitted, installed, selectable; current H3 status, 1KB, and 64KB validation and benchmarks passed in `job-610e9f2d38364cfc95b238ea6e012446` |
| `org.protocol-lab.components.implementation.quic-go-raw` | `0.1.1` | `93d7127beacc812de5042cc6b8f51d7681c6b6f378ead7dd730ab25f6e3d598c` | admitted, installed, selectable; live package-backed raw stream throughput and multiplex smoke passed in `job-43983c8c7a35400fa54767ba0be66045` |
| `org.protocol-lab.components.implementation.aioquic-http3` | `0.3.3` | `b9407f753cdb4275a36ca740836a28862925a56e64513312490c918674df0c83` | admitted, installed, selectable; current h3spec status, response-header, and QPACK diagnostics passed in `job-a3c8b35637e14c49b86332a928c5b15d`; canonical status and 1KB proof remains retained on immutable `0.3.2` in `job-0d08b2ace1704d609ec9803e6e7119c7` |
| `org.protocol-lab.components.implementation.kestrel-http3` | `0.1.6` | `2e526000048405a6cf3f3ebf22e9dba3b3d631a63cc18b3ffcc086ed881f7ae1` | admitted, installed, selectable; current H3 status, 1KB, and 64KB validation and benchmarks passed in `job-fb08e6a527b94ee1a922055a9401feee` |
| `org.protocol-lab.components.implementation.quiche-http3` | `0.1.5` | `870844a8ce4b36d975d3dcd4de1f12507a53720c395142605b7158ab8fa5c003` | admitted, installed, selectable; official H3 payload rows remain validation-failed because `content-type` is missing; `http3.external.peer-characterization` is the supported package-backed scenario, but `job-8f658e39aa0446a0a05240d580daed64` produced no completed live evidence |
| `org.protocol-lab.components.implementation.ngtcp2-http3` | `0.1.4` | `339340e1f1bccd3554a904ab661592749dd3922628f7f223492c9a6dbb04a67c` | admitted, installed, selectable; official H3 payload rows remain validation-failed because `content-type` is `text/plain`; `http3.external.peer-characterization` is the supported package-backed scenario, but `job-8f658e39aa0446a0a05240d580daed64` did not reach this cell before cancellation |
| `org.protocol-lab.components.executor.curl-http3-client` | `0.1.4` | `37b8d5227c67e2ebdb6e5ce7f0dc06ca492d2a1124b38b481719d9ecb13b4a88` | admitted, installed, selectable; supports `http3.external.peer-characterization` as diagnostic executor coverage |
| `org.protocol-lab.components.scenario.http3-peer-characterization` | `0.1.0` | `0c9d284493d6c2e39035535a6255411e495a135a957f555707e4fd740a7402e8` | admitted, installed, selectable; diagnostic scenario pack, not an official payload benchmark |
| `org.protocol-lab.components.executor.h3spec-http3-qpack` | `0.1.8` | `a97631a74a2c511b58b92e60f392b3f227d5e095264fd1f1e4dd610547c2aea1` | admitted, installed, selectable; exact identity and load-shape evidence passed in `job-a3c8b35637e14c49b86332a928c5b15d` |
| `org.protocol-lab.components.scenario.h3spec-http3-qpack` | `0.1.3` | `e89f6350ec7c0956431c1e7432f36b2df8daee8bccf7424edd9c562f6385786c` | admitted, installed, selectable; all three focused cells completed against aioquic `0.3.3` in `job-a3c8b35637e14c49b86332a928c5b15d` |

The controller also contains earlier immutable uploads from the Caddy, nginx, quic-go, aioquic, Kestrel, quiche, and ngtcp2 correction sequences. Do not use superseded or overbroad uploads for ranking. Use Caddy `0.1.7`, nginx `0.1.7`, quic-go HTTP/3 `0.1.6`, aioquic `0.3.3` for the current h3spec/QPACK rows, aioquic `0.3.2` for its retained payload rows, and Kestrel `0.1.6` only for the completed rows cited above. Quiche `0.1.5`, ngtcp2 `0.1.4`, curl `0.1.4`, and `http3-peer-characterization` `0.1.0` are admitted/selectable, but their attempted characterization job produced no completed evidence.

## Remaining Ranked Gaps

| Rank | Gap | Value | Blocker |
| --- | --- | --- | --- |
| 1 | Package-backed `quic-dotnet-dev` HTTP/3 implementation handoff in controller inventory | Highest visible first-party HTTP/3 lane | Owned outside this repository |
| 2 | Package-backed `quic-dotnet-raw-dev` or MSQuic raw QUIC implementation handoff | First-party raw QUIC lane alongside the new quic-go ecosystem peer | Owned outside this repository |
| 3 | Caddy, nginx, quic-go, Kestrel, and aioquic HTTP/3 large-body/header-heavy fixtures | Makes public comparison rows richer | Needs broader live controller validation and deterministic fixture behavior beyond the completed rows cited above |
| 4 | quic-go HTTP/3 client package | Useful ecosystem peer | Current package is server-only |
| 5 | xquic HTTP/3 package | Additional ecosystem peer diversity | Deferred until local peer stability and acquisition are proven |
