# ProtocolLab Package Coverage Matrix

Observed: 2026-06-20

This matrix tracks reusable component package coverage for implementations and executors that should be visible in ProtocolLab comparisons. It separates package inventory from proof quality: `supported` means the package metadata and wrapper advertise the row; live benchmark or conformance evidence remains tied to report artifacts.

## Inputs

| Source | Observed entries |
| --- | --- |
| Component packages before this change | `aioquic-http3`, `quiche-http3`, `ngtcp2-http3`, `kestrel-http3`, `quic-go-raw-load`, `h3spec-http3-qpack`, `aioquic-rfc9220-websocket`, plus HTTP/1 packages and the raw QUIC scenario pack |
| Public site implementation catalog | `kestrel-http3`, `incursa-http3`, `msquic-dotnet`, `caddy-http3`, `nginx-http3` package-backed entry |
| Live controller inventory before registration | 184 packages, 103 implementations, 16 test executors, 58 scenarios, 14 suites, 22 known package IDs |
| Live controller inventory after final registration | 187 packages, 106 implementations, 16 test executors, 58 scenarios, 14 suites, 23 known package IDs |
| Live controller known component IDs before registration | `aioquic-http3`, `quiche-http3`, `ngtcp2-http3`, `kestrel-http1`, `quic-go-raw-load`, `curl-http3-client`, `h3spec-http3-qpack`, `aioquic-rfc9220-websocket`, `raw-quic-transport` |
| Local source-context implementations | `quic-dotnet-dev`, `quic-dotnet-raw-dev`, `msquic-dotnet-raw-adapter-v1`, `incursa-http3` remain implementation-owned outside this repository |

## Before And After

| Visible implementation or executor | Before package state | After package state |
| --- | --- | --- |
| `kestrel-http3` | source package present | source package present with explicit scenario coverage metadata |
| `caddy-http3` | visible on public site and present in internal consumer manifests, but not packaged in this repo | added and registered `org.protocol-lab.components.implementation.caddy-http3` |
| `nginx-http3` | present in internal consumer manifests, but not packaged in this repo | added `org.protocol-lab.components.implementation.nginx-http3` with nginx `-V` HTTP/3 module proof |
| `incursa-http3` / `quic-dotnet-dev` | live/package-owned outside this repo | unchanged; not duplicated in component repo |
| `msquic-dotnet` / `quic-dotnet-raw-dev` | live/package-owned outside this repo | unchanged; raw QUIC scenario and quic-go executor remain reusable component packages |
| `aioquic-http3` | source package present | source package present with explicit scenario coverage metadata |
| `quiche-http3` | source package present | source package present with explicit scenario coverage metadata |
| `ngtcp2-http3` | source package present | source package present with explicit scenario coverage metadata |
| `quic-go-raw-load` | source executor package present | source executor package present with explicit comparison-lane coverage metadata |
| `h3spec-http3-qpack` | source executor package present | source executor package present with explicit comparison-lane coverage metadata |
| `aioquic-rfc9220-websocket` | source executor package present | source executor package present with explicit comparison-lane coverage metadata |

## Scenario Support

| Package | Raw QUIC | H3 1KB | H3 64KB | H3 large body | Header-heavy / QPACK | WebSocket-over-H3 | h3spec / QPACK executor |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `kestrel-http3` | unsupported | supported | supported | supported | unsupported | unsupported | compatible target, unproven |
| `caddy-http3` | unsupported | supported | supported | unsupported | unsupported | unsupported | compatible target, unproven |
| `nginx-http3` | unsupported | supported | supported | skipped until live proof | unsupported | unsupported | compatible target, unproven |
| `aioquic-http3` | unsupported | partial | unproven | unsupported | supported metadata | executor-only via separate package | compatible target, unproven |
| `quiche-http3` | unsupported | partial | supported | supported | client-only | unsupported | compatible target, unproven |
| `ngtcp2-http3` | unsupported | partial | supported | supported | client-only | unsupported | compatible target, unproven |
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
| `org.protocol-lab.components.implementation.kestrel-http3` | `implementations/kestrel-http3` |
| `org.protocol-lab.components.executor.quic-go-raw-load` | `executors/quic-go-raw-load` |
| `org.protocol-lab.components.executor.h3spec-http3-qpack` | `executors/h3spec-http3-qpack` |
| `org.protocol-lab.components.executor.aioquic-rfc9220-websocket` | `executors/aioquic-rfc9220-websocket` |

## Registered Package

| Package | Version | SHA-256 | Controller status |
| --- | --- | --- | --- |
| `org.protocol-lab.components.implementation.caddy-http3` | `0.1.2` | `c427787beb24b946c4152ee0c6ff21ac97d2a19ed4ff9915adcc7026dce20b52` | admitted, installed, selectable |
| `org.protocol-lab.components.implementation.nginx-http3` | `0.1.1` | `d48b1121b2121266eff5c2d54876c83f72161f11a83e92d73b09b5574f6ee501` | admitted, installed, selectable; live package-backed job completed unsupported because worker image was not present and controller jobs do not forward target Docker build |

The controller also contains earlier immutable `0.1.0` and `0.1.1` uploads for `org.protocol-lab.components.implementation.caddy-http3` from the Caddy registration attempt sequence. Use Caddy `0.1.2` as the final Caddy package version from that repository state.

## Remaining Ranked Gaps

| Rank | Gap | Value | Blocker |
| --- | --- | --- | --- |
| 1 | Package-backed `quic-dotnet-dev` HTTP/3 implementation handoff in controller inventory | Highest visible first-party HTTP/3 lane | Owned outside this repository |
| 2 | Package-backed `quic-dotnet-raw-dev` or MSQuic raw QUIC implementation handoff | Highest visible raw QUIC lane | Owned outside this repository |
| 3 | Caddy and nginx HTTP/3 large-body/header-heavy fixtures | Makes public comparison rows richer | Needs live controller validation and deterministic fixture behavior |
| 4 | quic-go HTTP/3 server/client package | Useful ecosystem peer | Internal manifest is placeholder-only; no locally runnable package source |
| 5 | xquic HTTP/3 package | Additional ecosystem peer diversity | Deferred until local peer stability and acquisition are proven |
