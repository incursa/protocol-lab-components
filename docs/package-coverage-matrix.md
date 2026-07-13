# ProtocolLab Package Coverage Matrix

Observed: 2026-07-13

This matrix tracks reusable component package coverage for implementations and executors that should be visible in ProtocolLab comparisons. It separates package inventory from proof quality: `supported` means the package metadata and wrapper advertise the row; live benchmark or conformance evidence remains tied to report artifacts.

## Inputs

| Source | Observed entries |
| --- | --- |
| Component packages before this change | `aioquic-http3`, `quiche-http3`, `ngtcp2-http3`, `kestrel-http3`, `quic-go-http3`, `quic-go-raw-load`, `h3spec-http3-qpack`, `aioquic-rfc9220-websocket`, plus HTTP/1 packages and the raw QUIC scenario pack |
| Scenario packs added for controller selection | `h3spec-http3-qpack`, `http3-peer-characterization`, and `aioquic-rfc9220-websocket` focused scenario packs |
| Public site implementation catalog | `kestrel-http3`, `incursa-http3`, `msquic-dotnet`, `caddy-http3`, `nginx-http3`, and planned `quic-go-http3` entry |
| Live controller inventory after Caddy/nginx correction | `org.protocol-lab.components.implementation.caddy-http3` and `org.protocol-lab.components.implementation.nginx-http3` versions through final `0.1.7` are installed/selectable |
| Live controller final quic-go proof | `job-020c0660877243b0b970578c139aefe2`; H3 1KB validation passed, benchmark succeeded, package-backed provenance recorded |
| Live controller final quic-go raw proof | `job-43983c8c7a35400fa54767ba0be66045`; raw stream throughput and multiplex validation passed, benchmark succeeded, package-backed provenance recorded |
| Live controller final Caddy proof | `job-c92b1918b59846018dcc808babd58730`; H3 status, 1KB, and 64KB validation passed, benchmark succeeded, package-backed provenance recorded |
| Live controller final Caddy/nginx correction proof | `job-e30007af97354306ab7d34abbbbbab4a`; Caddy and nginx H3 status, 1KB, and 64KB validation passed, benchmark succeeded with final `0.1.7` packages; controller-host publication lacked `CLOUDFLARE_API_TOKEN`, so the staged bundle was uploaded with local Wrangler auth, queued, imported into D1, and appeared on live 1KB/64KB leaderboards |
| Live controller peer characterization attempt | `job-8f658e39aa0446a0a05240d580daed64`; preview selected two runnable diagnostic cells for quiche `0.1.5` and ngtcp2 `0.1.4`, but the job was cancelled after remaining in `running-benchmark` on quiche cell 1 with zero completed cells |
| Local source-context implementations | `quic-dotnet-dev`, `quic-dotnet-raw-dev`, `msquic-dotnet-raw-adapter-v1`, `incursa-http3` remain implementation-owned outside this repository |

## Before And After

| Visible implementation or executor | Before package state | After package state |
| --- | --- | --- |
| `kestrel-http3` | source package present | source package present with explicit scenario coverage metadata |
| `caddy-http3` | visible on public site and present in internal consumer manifests, but not packaged in this repo | added and registered `org.protocol-lab.components.implementation.caddy-http3` |
| `nginx-http3` | present in internal consumer manifests, but not packaged in this repo | added `org.protocol-lab.components.implementation.nginx-http3`; live H3 status, 1KB, and 64KB smoke passed |
| `incursa-http3` / `quic-dotnet-dev` | live/package-owned outside this repo | unchanged; not duplicated in component repo |
| `msquic-dotnet` / `quic-dotnet-raw-dev` | live/package-owned outside this repo | unchanged; raw QUIC scenario and quic-go executor remain reusable component packages |
| `aioquic-http3` | source package present | source package present with explicit scenario coverage metadata |
| `quiche-http3` | source package present | source package present with explicit scenario coverage metadata |
| `ngtcp2-http3` | source package present | source package present with explicit scenario coverage metadata |
| `quic-go-http3` | protocol-lab-internal placeholder and raw QUIC executor only | added `org.protocol-lab.components.implementation.quic-go-http3`; live H3 1KB smoke passed |
| `quic-go-raw` | no raw QUIC implementation package | added `org.protocol-lab.components.implementation.quic-go-raw` for `quic.transport.stream-throughput.1mb` and `quic.transport.multiplex.100x64kb` only |
| `quic-go-raw-load` | source executor package present | source executor package present with explicit comparison-lane coverage metadata |
| `h3spec-http3-qpack` | source executor package present | source executor package present with explicit comparison-lane coverage metadata |
| `aioquic-rfc9220-websocket` | source executor package present | source executor package present with explicit comparison-lane coverage metadata |
| `h3spec-http3-qpack` scenario pack | missing | added `org.protocol-lab.components.scenario.h3spec-http3-qpack` with suite `h3spec-http3-qpack-focused` bound to `h3spec-http3-qpack` |
| `http3-peer-characterization` scenario pack | missing | added `org.protocol-lab.components.scenario.http3-peer-characterization` with suite `http3-peer-characterization` for diagnostic external peer rows |
| `aioquic-rfc9220-websocket` scenario pack | missing | added `org.protocol-lab.components.scenario.aioquic-rfc9220-websocket` with suite `aioquic-rfc9220-websocket-proof` bound to `aioquic-rfc9220-websocket` |
| `go-http1-executor` | missing | local diagnostic package `org.protocol-lab.components.executor.go-http1-executor@0.3.0`; exact HTTP/1.1 validation plus pinned `oha@1.15.0` load generation |
| `go-http2-executor` | missing | local diagnostic package `org.protocol-lab.components.executor.go-http2-executor@0.3.0`; exact h2c smoke, diagnostic, and `16/128/8` balanced-round-robin comparison shapes |
| `go-tls13-executor` | missing | local diagnostic package `org.protocol-lab.components.executor.go-tls13-executor@0.3.0`; exact TLS 1.3 full/resumed handshakes plus deterministic record throughput and six-case record coverage; remaining committed TLS identities fail closed as explicit `unsupported` |
| `dotnet-sslstream-tls13` | missing | local diagnostic package `org.protocol-lab.components.implementation.dotnet-sslstream-tls13@0.1.0`; independent library-backed TLS 1.3 target |
| `go-tls13` | missing | local diagnostic package `org.protocol-lab.components.implementation.go-tls13@0.2.0`; independent Go `crypto/tls` target for exact TLS 1.3 full/resumed handshakes and deterministic record transfers |
| HTTP/1.1 cleartext WebSocket three-package lane | missing | local diagnostic scenario, executor, and independent origin packages at `0.1.0`; exact five-ID RFC 6455 smoke passes, while all adjacent WebSocket identities fail closed as unsupported |

## Scenario Support

| Package | Raw QUIC | H3 1KB | H3 64KB | H3 large body | Header-heavy / QPACK | WebSocket-over-H3 | h3spec / QPACK executor |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `kestrel-http3` | unsupported | supported | supported | supported | unsupported | unsupported | compatible target, unproven |
| `caddy-http3` | unsupported | supported, live proof | supported, live proof | unsupported | unsupported | unsupported | compatible target, unproven |
| `nginx-http3` | unsupported | supported, live proof | supported, live proof | skipped pending broader fixture promotion | unsupported | unsupported | compatible target, unproven |
| `aioquic-http3` | unsupported | supported | unproven | unsupported | supported metadata | supported via RFC9220 endpoint and executor | compatible target, unproven |
| `quiche-http3` | unsupported | validation-failed; characterization supported | validation-failed; characterization supported | validation-failed; characterization supported | client-only | unsupported | compatible target, unproven |
| `ngtcp2-http3` | unsupported | validation-failed; characterization supported | validation-failed; characterization supported | validation-failed; characterization supported | client-only | unsupported | compatible target, unproven |
| `quic-go-http3` | unsupported | supported, live proof | supported metadata | supported metadata | unsupported | unsupported | compatible target, unproven |
| `quic-go-raw` | supported target for stream throughput and multiplex only, live proof | unsupported | unsupported | unsupported | unsupported | unsupported | unsupported |
| `quic-go-raw-load` | supported executor | unsupported | unsupported | unsupported | unsupported | unsupported | unsupported |
| `h3spec-http3-qpack` | unsupported | compatible target, unproven | compatible target, unproven | compatible target, unproven | supported | unsupported | supported |
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
| `org.protocol-lab.components.executor.go-tls13-executor` | `executors/go-tls13-executor` |
| `org.protocol-lab.components.implementation.dotnet-sslstream-tls13` | `implementations/dotnet-sslstream-tls13` |
| `org.protocol-lab.components.implementation.go-tls13` | `implementations/go-tls13` |
| `org.protocol-lab.components.implementation.go-http1-websocket` | `implementations/go-http1-websocket` |
| `org.protocol-lab.components.executor.go-http1-websocket-executor` | `executors/go-http1-websocket-executor` |
| `org.protocol-lab.components.scenario.http1-websocket-cleartext-performance` | `scenarios/http1-websocket-cleartext-performance` |
| `org.protocol-lab.components.scenario.raw-quic-transport` | `scenarios/raw-quic-transport` |
| `org.protocol-lab.components.scenario.h3spec-http3-qpack` | `scenarios/h3spec-http3-qpack` |
| `org.protocol-lab.components.scenario.http3-peer-characterization` | `scenarios/http3-peer-characterization` |
| `org.protocol-lab.components.scenario.aioquic-rfc9220-websocket` | `scenarios/aioquic-rfc9220-websocket` |

## Registered Package

| Package | Version | SHA-256 | Controller status |
| --- | --- | --- | --- |
| `org.protocol-lab.components.implementation.caddy-http3` | `0.1.7` | `96b3c8d1859ef927dcaee0866d4b5b73e35f5a4d61a897a2d35dc8d59f28fa2d` | admitted, installed, selectable; live package-backed H3 status, 1KB, and 64KB smoke passed in `job-e30007af97354306ab7d34abbbbbab4a`; 1MB remains unsupported until package-backed live proof covers it |
| `org.protocol-lab.components.implementation.nginx-http3` | `0.1.7` | `a1bc60a99a5a8c96684d9ede91b93a0b393f6174b89d9bc8d103759e8d6a9175` | admitted, installed, selectable; live package-backed H3 status, 1KB, and 64KB smoke passed in `job-e30007af97354306ab7d34abbbbbab4a`; 1MB remains skipped/not declared until package-backed live proof covers it |
| `org.protocol-lab.components.implementation.quic-go-http3` | `0.1.4` | `c8ab6aad32280abd3da20ce5c6a0422eb8c23147c9423387b274150ffba534a1` | admitted, installed, selectable; prior live package-backed H3 1KB smoke passed with `0.1.3` in `job-020c0660877243b0b970578c139aefe2`; rerun `0.1.4` before claiming updated live proof |
| `org.protocol-lab.components.implementation.quic-go-raw` | `0.1.1` | `93d7127beacc812de5042cc6b8f51d7681c6b6f378ead7dd730ab25f6e3d598c` | admitted, installed, selectable; live package-backed raw stream throughput and multiplex smoke passed in `job-43983c8c7a35400fa54767ba0be66045` |
| `org.protocol-lab.components.implementation.aioquic-http3` | `0.1.7` | `18c3214b6292756ea5222ae80270b81703218f3644ffd85240bfa1b84690dd02` | admitted, installed, selectable; earlier live package-backed H3 1KB smoke passed with `0.1.6` in `scenario-declarations-aioquic-1kb-final-h3-local-v1`; rerun `0.1.7` before claiming updated live proof |
| `org.protocol-lab.components.implementation.quiche-http3` | `0.1.5` | `870844a8ce4b36d975d3dcd4de1f12507a53720c395142605b7158ab8fa5c003` | admitted, installed, selectable; official H3 payload rows remain validation-failed because `content-type` is missing; `http3.external.peer-characterization` is the supported package-backed scenario, but `job-8f658e39aa0446a0a05240d580daed64` produced no completed live evidence |
| `org.protocol-lab.components.implementation.ngtcp2-http3` | `0.1.4` | `339340e1f1bccd3554a904ab661592749dd3922628f7f223492c9a6dbb04a67c` | admitted, installed, selectable; official H3 payload rows remain validation-failed because `content-type` is `text/plain`; `http3.external.peer-characterization` is the supported package-backed scenario, but `job-8f658e39aa0446a0a05240d580daed64` did not reach this cell before cancellation |
| `org.protocol-lab.components.executor.curl-http3-client` | `0.1.4` | `37b8d5227c67e2ebdb6e5ce7f0dc06ca492d2a1124b38b481719d9ecb13b4a88` | admitted, installed, selectable; supports `http3.external.peer-characterization` as diagnostic executor coverage |
| `org.protocol-lab.components.scenario.http3-peer-characterization` | `0.1.0` | `0c9d284493d6c2e39035535a6255411e495a135a957f555707e4fd740a7402e8` | admitted, installed, selectable; diagnostic scenario pack, not an official payload benchmark |

The controller also contains earlier immutable `0.1.0`, `0.1.1`, `0.1.2`, `0.1.4`, `0.1.5`, and overbroad `0.1.6` uploads for `org.protocol-lab.components.implementation.caddy-http3`, earlier nginx `0.1.1`, `0.1.3`, `0.1.4`, `0.1.5`, and overbroad `0.1.6` uploads, `0.1.0`, `0.1.1`, `0.1.2`, and `0.1.3` uploads for `org.protocol-lab.components.implementation.quic-go-http3` from the process/docker correction sequence, an earlier `0.1.0` upload for `org.protocol-lab.components.implementation.quic-go-raw` with the wrong readiness token, earlier `0.1.4`, `0.1.5`, and `0.1.6` uploads for `org.protocol-lab.components.implementation.aioquic-http3`, earlier `0.1.1`, `0.1.2`, and `0.1.3` uploads for `org.protocol-lab.components.implementation.quiche-http3`, and earlier `0.1.1` and `0.1.2` uploads for `org.protocol-lab.components.implementation.ngtcp2-http3`. Do not use the Caddy/nginx `0.1.6` uploads for ranking: they declared 1MB but the live worker could not prove those rows because the matching target images were not available. Use Caddy `0.1.7` and nginx `0.1.7` for the Caddy/nginx rows from this pass; quic-go HTTP/3 `0.1.4`, aioquic `0.1.7`, quiche `0.1.5`, ngtcp2 `0.1.4`, curl `0.1.4`, and `http3-peer-characterization` `0.1.0` are admitted/selectable, but only the rows with completed jobs above have live proof.

## Remaining Ranked Gaps

| Rank | Gap | Value | Blocker |
| --- | --- | --- | --- |
| 1 | Package-backed `quic-dotnet-dev` HTTP/3 implementation handoff in controller inventory | Highest visible first-party HTTP/3 lane | Owned outside this repository |
| 2 | Package-backed `quic-dotnet-raw-dev` or MSQuic raw QUIC implementation handoff | First-party raw QUIC lane alongside the new quic-go ecosystem peer | Owned outside this repository |
| 3 | Caddy, nginx, and quic-go HTTP/3 large-body/header-heavy fixtures | Makes public comparison rows richer | Needs broader live controller validation and deterministic fixture behavior beyond Caddy H3 status/1KB/64KB and quic-go H3 1KB |
| 4 | quic-go HTTP/3 client package | Useful ecosystem peer | Current package is server-only |
| 5 | xquic HTTP/3 package | Additional ecosystem peer diversity | Deferred until local peer stability and acquisition are proven |
