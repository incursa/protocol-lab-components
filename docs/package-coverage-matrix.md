# ProtocolLab Package Coverage Matrix

Observed: 2026-06-20

This matrix tracks reusable component package coverage for implementations and executors that should be visible in ProtocolLab comparisons. It separates package inventory from proof quality: `supported` means the package metadata and wrapper advertise the row; live benchmark or conformance evidence remains tied to report artifacts.

## Inputs

| Source | Observed entries |
| --- | --- |
| Component packages before this change | `aioquic-http3`, `quiche-http3`, `ngtcp2-http3`, `kestrel-http3`, `quic-go-raw-load`, `h3spec-http3-qpack`, `aioquic-rfc9220-websocket`, plus HTTP/1 packages and the raw QUIC scenario pack |
| Scenario packs added for controller selection | `h3spec-http3-qpack` and `aioquic-rfc9220-websocket` focused scenario packs |
| Public site implementation catalog | `kestrel-http3`, `incursa-http3`, `msquic-dotnet`, `caddy-http3`, `nginx-http3`, and planned `quic-go-http3` entry |
| Live controller inventory after Caddy publication | 214 package records; `org.protocol-lab.components.implementation.caddy-http3` versions through final `0.1.4` are installed/selectable |
| Live controller final quic-go proof | `job-020c0660877243b0b970578c139aefe2`; H3 1KB validation passed, benchmark succeeded, package-backed provenance recorded |
| Live controller final Caddy proof | `job-c92b1918b59846018dcc808babd58730`; H3 status, 1KB, and 64KB validation passed, benchmark succeeded, package-backed provenance recorded |
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
| `quic-go-raw-load` | source executor package present | source executor package present with explicit comparison-lane coverage metadata |
| `h3spec-http3-qpack` | source executor package present | source executor package present with explicit comparison-lane coverage metadata |
| `aioquic-rfc9220-websocket` | source executor package present | source executor package present with explicit comparison-lane coverage metadata |
| `h3spec-http3-qpack` scenario pack | missing | added `org.protocol-lab.components.scenario.h3spec-http3-qpack` with suite `h3spec-http3-qpack-focused` bound to `h3spec-http3-qpack` |
| `aioquic-rfc9220-websocket` scenario pack | missing | added `org.protocol-lab.components.scenario.aioquic-rfc9220-websocket` with suite `aioquic-rfc9220-websocket-proof` bound to `aioquic-rfc9220-websocket` |

## Scenario Support

| Package | Raw QUIC | H3 1KB | H3 64KB | H3 large body | Header-heavy / QPACK | WebSocket-over-H3 | h3spec / QPACK executor |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `kestrel-http3` | unsupported | supported | supported | supported | unsupported | unsupported | compatible target, unproven |
| `caddy-http3` | unsupported | supported, live proof | supported, live proof | unsupported | unsupported | unsupported | compatible target, unproven |
| `nginx-http3` | unsupported | supported, live proof | supported, live proof | skipped pending broader fixture promotion | unsupported | unsupported | compatible target, unproven |
| `aioquic-http3` | unsupported | supported | unproven | unsupported | supported metadata | executor-only via separate package | compatible target, unproven |
| `quiche-http3` | unsupported | validation-failed | supported | supported | client-only | unsupported | compatible target, unproven |
| `ngtcp2-http3` | unsupported | validation-failed | supported | supported | client-only | unsupported | compatible target, unproven |
| `quic-go-http3` | unsupported | supported, live proof | supported metadata | supported metadata | unsupported | unsupported | compatible target, unproven |
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
| `org.protocol-lab.components.implementation.kestrel-http3` | `implementations/kestrel-http3` |
| `org.protocol-lab.components.executor.quic-go-raw-load` | `executors/quic-go-raw-load` |
| `org.protocol-lab.components.executor.h3spec-http3-qpack` | `executors/h3spec-http3-qpack` |
| `org.protocol-lab.components.executor.aioquic-rfc9220-websocket` | `executors/aioquic-rfc9220-websocket` |
| `org.protocol-lab.components.scenario.raw-quic-transport` | `scenarios/raw-quic-transport` |
| `org.protocol-lab.components.scenario.h3spec-http3-qpack` | `scenarios/h3spec-http3-qpack` |
| `org.protocol-lab.components.scenario.aioquic-rfc9220-websocket` | `scenarios/aioquic-rfc9220-websocket` |

## Registered Package

| Package | Version | SHA-256 | Controller status |
| --- | --- | --- | --- |
| `org.protocol-lab.components.implementation.caddy-http3` | `0.1.4` | `cf5d9af7e4daa09fadf2e2a74387b8f29c994764f5cc6bf3afa8ced99576783d` | admitted, installed, selectable; live package-backed H3 status, 1KB, and 64KB smoke passed in `job-c92b1918b59846018dcc808babd58730` |
| `org.protocol-lab.components.implementation.nginx-http3` | `0.1.4` | `f7a29eaeb8060d10d02a28df53428c501e446678bbe4524c360da190b1056ef5` | admitted, installed, selectable; live package-backed H3 status, 1KB, and 64KB smoke passed in `job-02ecc4eb59124657b13ea0f9d2bbd428` |
| `org.protocol-lab.components.implementation.quic-go-http3` | `0.1.3` | `8823bf16784e017ab4c953e0232dc6e618d3fd19b707322582d097c02d6d0f55` | admitted, installed, selectable; live package-backed H3 1KB smoke passed in `job-020c0660877243b0b970578c139aefe2` |
| `org.protocol-lab.components.implementation.aioquic-http3` | `0.1.6` | `42364df528eec501862ac349bc11124fbd7b38ea0f25d2bb6a27e7390f9b9de6` | admitted, installed, selectable; live package-backed H3 1KB smoke passed in `scenario-declarations-aioquic-1kb-final-h3-local-v1` |
| `org.protocol-lab.components.implementation.quiche-http3` | `0.1.3` | `cfdf7e506712497032bd7614c32f7bfe7bef5634a383257ead004d92460f512b` | admitted, installed, selectable; live package-backed H3 1KB smoke is validation-failed in `scenario-declarations-quiche-1kb-final-h3-local-v1` because `content-type` is missing |
| `org.protocol-lab.components.implementation.ngtcp2-http3` | `0.1.2` | `84fcb9cfbffe6ca9e0b1faa5876d49196deb096ca0fc8ecc34689bbc261dcdbb` | admitted, installed, selectable; live package-backed H3 1KB smoke is validation-failed in `scenario-declarations-ngtcp2-1kb-final-h3-local-v1` because `content-type` is `text/plain` |

The controller also contains earlier immutable `0.1.0`, `0.1.1`, and `0.1.2` uploads for `org.protocol-lab.components.implementation.caddy-http3` from the Caddy registration attempt sequence, earlier nginx `0.1.1` and `0.1.3` uploads from the pre-Docker-worker setup sequence, `0.1.0`, `0.1.1`, and `0.1.2` uploads for `org.protocol-lab.components.implementation.quic-go-http3` from the process/docker correction sequence, earlier `0.1.4` and `0.1.5` uploads for `org.protocol-lab.components.implementation.aioquic-http3`, earlier `0.1.1` and `0.1.2` uploads for `org.protocol-lab.components.implementation.quiche-http3`, and earlier `0.1.1` uploads for `org.protocol-lab.components.implementation.ngtcp2-http3`. Use Caddy `0.1.4`, nginx `0.1.4`, quic-go `0.1.3`, aioquic `0.1.6`, quiche `0.1.3`, and ngtcp2 `0.1.2` as the final package versions from this repository state.

## Remaining Ranked Gaps

| Rank | Gap | Value | Blocker |
| --- | --- | --- | --- |
| 1 | Package-backed `quic-dotnet-dev` HTTP/3 implementation handoff in controller inventory | Highest visible first-party HTTP/3 lane | Owned outside this repository |
| 2 | Package-backed `quic-dotnet-raw-dev` or MSQuic raw QUIC implementation handoff | Highest visible raw QUIC lane | Owned outside this repository |
| 3 | Caddy, nginx, and quic-go HTTP/3 large-body/header-heavy fixtures | Makes public comparison rows richer | Needs broader live controller validation and deterministic fixture behavior beyond Caddy H3 status/1KB/64KB and quic-go H3 1KB |
| 4 | quic-go HTTP/3 client package | Useful ecosystem peer | Current package is server-only |
| 5 | xquic HTTP/3 package | Additional ecosystem peer diversity | Deferred until local peer stability and acquisition are proven |
