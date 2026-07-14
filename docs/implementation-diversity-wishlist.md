# ProtocolLab Implementation Diversity Wishlist

Updated: 2026-07-13

This is the delivery backlog for expanding ProtocolLab's real implementation
testing surface after the public contracts, scenarios, and test executors have
been bootstrapped. The program favors wrappers around unmodified upstream
software. Thin fixture adapters are allowed only where the protocol requires an
application-defined service or echo endpoint, and those adapters must implement
the existing ProtocolLab contract exactly.

The immediate priority is HTTP/2 implementation diversity. ProtocolLab already
has broad HTTP/3 coverage, while its general-purpose HTTP/2 origin coverage is
currently centered on Kestrel.

## Program rules

- Keep HTTP/1.1, HTTP/2 h2c, HTTP/2 TLS/ALPN, HTTP/3, TLS endpoint, secure-DNS,
  gRPC, and WebSocket identities explicit. Do not silently reuse a package from
  another protocol lane.
- Wrap upstream programs without modifying their server implementations.
- Keep public metadata in `protocol-lab-package.json` and execution-only
  acquisition, entrypoint, runtime, and container details in
  `protocol-lab.internal.json`, scripts, or package-local documentation.
- Declare only scenarios whose exact semantics the target can satisfy. Record
  unsupported rows, with reasons, instead of approximating upload, hashing,
  streaming, or content behavior.
- Separate production-origin, diagnostic-tool, authoritative-DNS, resolver,
  and diagnostic-WebSocket cohorts. Do not rank unlike roles together.
- Treat each `.plabpkg` as an immutable run input and preserve the selected
  scenario package, test-executor package, implementation package, versions,
  hashes, raw stdout/stderr, validation result, and report bundle in evidence.
- Do not claim live support from manifest validation or a plan-only smoke.

## Definition of done for each runnable package

1. The package has unique public and internal identities, exact scenario
   declarations, a focused README, and a standard repository build entrypoint.
2. Repository manifest validation and package-specific tests pass.
3. The `.plabpkg` and matching build attestation are produced and their
   identity, version, runtime, source commit, and SHA-256 agree.
4. The built archive is materialized and passes its package-local smoke.
5. The real ProtocolLab controller admits and materializes the immutable
   package on a compatible worker.
6. A compatible existing executor runs the declared scenario against the live
   implementation. If no compatible executor exists, the missing executor is
   an explicit prerequisite rather than a reason to fake proof.
7. Validation passes before benchmark evidence is accepted. Raw tool output and
   the resulting evidence bundle are retained, and unsupported/unavailable
   cells remain explicit.

## Wave 1 - general-purpose HTTP origins

The target matrix is four comparable server families across HTTP/1.1 and
HTTP/2. HTTP/2 packages must expose h2c prior-knowledge and TLS/ALPN as distinct
selectable execution variants when upstream supports both.

| Item | Package identity | Initial exact scope | Acceptance target |
| --- | --- | --- | --- |
| Caddy HTTP/2 | `org.protocol-lab.components.implementation.caddy-http2` | Config-backed h2c and TLS/ALPN; plaintext/static payload, JSON fixture, download, headers, and connection reuse rows that pass exact validation | Run with the HTTP/2 executor on real ProtocolLab in both declared variants |
| nginx HTTP/2 | `org.protocol-lab.components.implementation.nginx-http2` | Distinct wrapper reusing nginx conventions; h2c first where compatible, plus separate TLS/ALPN variant | Run with exact HTTP/2 proof; never inherit HTTP/1 or HTTP/3 evidence |
| Apache HTTP/1.1 | `org.protocol-lab.components.implementation.apache-http1` | Config-backed response, headers, deterministic static payload/JSON fixture, downloads, and reuse | Add the fourth comparable HTTP/1.1 origin family and run it with the HTTP/1 executor |
| Apache HTTP/2 | `org.protocol-lab.components.implementation.apache-http2` | `mod_http2` h2c and TLS `h2` variants for configuration-backed response and transfer scenarios | Run both declared variants with HTTP/2 proof |

Apache CGI and custom modules are out of scope for the initial wrapper. Upload
processing, hashing, and application streaming stay unsupported until an
unmodified upstream module satisfies the exact semantics.

## Wave 2 - TLS endpoint/tool cohort

These packages are TLS diagnostic endpoints, not application servers, and must
be compared in their own cohort.

| Item | Package identity | Initial exact scope | Follow-up controls |
| --- | --- | --- | --- |
| OpenSSL `s_server` | `org.protocol-lab.components.implementation.openssl-s-server` | TLS 1.3 full and resumed handshakes, record transfer, compatible mTLS | ALPN, session cache, cipher selection, early data, and KeyUpdate diagnostics where the executable exposes exact controls |
| GnuTLS `gnutls-serv` | `org.protocol-lab.components.implementation.gnutls-serv` | Handshake and record-transfer rows with explicit ALPN, certificates, client authentication, priority strings, and record sizing | Add only rows proven against the TLS contracts and executor |
| rustls upstream example server | Identity to finalize after upstream-control audit | Wrapper-only candidate | Admit only if the unmodified program exposes every required contract control |
| s2n-tls upstream server utility | Identity to finalize after upstream-control audit | Wrapper-only candidate | Admit only if the unmodified program exposes every required contract control |

## Wave 3 - secure DNS authoritative cohort

All authoritative targets host the deterministic local `plab.test` zone with
no recursive upstream dependency. Transport lanes remain separate package
identities even when one upstream server provides all of them.

| Item | Package identity | Required lane |
| --- | --- | --- |
| Technitium DoT | `org.protocol-lab.components.implementation.technitium-dot` | DNS over TLS |
| Technitium DoH2 | `org.protocol-lab.components.implementation.technitium-doh2` | DNS over HTTPS using HTTP/2 |
| Technitium DoH3 | `org.protocol-lab.components.implementation.technitium-doh3` | DNS over HTTPS using HTTP/3 |
| Technitium DoQ | `org.protocol-lab.components.implementation.technitium-doq` | DNS over QUIC |
| BIND 9 DoT | `org.protocol-lab.components.implementation.bind9-dot` | Incoming authoritative DNS over TLS |
| BIND 9 DoH2 | `org.protocol-lab.components.implementation.bind9-doh2` | Incoming authoritative DNS over HTTPS using HTTP/2 |

Before bundling Technitium binaries or images, record a redistribution decision
for its GPL-3.0 licensing. A source/config-only wrapper or externally acquired
runtime remains acceptable when redistribution is not approved.

## Wave 4 - secure DNS resolver cohort

Add Unbound DoT and DoH2 wrappers after the authoritative matrix works. Unbound
is a recursive validating caching resolver and must use separate package IDs,
scenarios, comparison groups, and ranking policy from Technitium, BIND, and the
existing authoritative fixture target. Final package identities are chosen when
the resolver scenarios are audited against the public contracts.

## Wave 5 - gRPC runtime diversity

gRPC cannot be a configuration-only wrapper because the ProtocolLab protobuf
service must be hosted. The approved implementation strategy is a minimal,
parity-tested adapter around each existing framework: generated bindings plus
only the service glue required by the canonical contract.

| Runtime | Working package identity | Required outcome |
| --- | --- | --- |
| ASP.NET Core / grpc-dotnet | `org.protocol-lab.components.implementation.grpc-dotnet` | Canonical unary and streaming service rows executed by the existing gRPC executor |
| gRPC Java / Netty | `org.protocol-lab.components.implementation.grpc-java-netty` | Same contract and fixture semantics, with runtime identity preserved |
| gRPC C++ | `org.protocol-lab.components.implementation.grpc-cpp` | Same contract and fixture semantics, with native build/runtime proof |
| Node `grpc-js` | `org.protocol-lab.components.implementation.grpc-js` | Later cross-platform representative after the first three reach parity |

An upstream benchmark server with a different protobuf contract is not valid
evidence and must not be substituted.

## Wave 6 - WebSocket diversity

| Item | Working package identity | Role and boundary |
| --- | --- | --- |
| websocat HTTP/1 WebSocket | `org.protocol-lab.components.implementation.websocat-http1-websocket` | Diagnostic mirror baseline only; no RFC 8441 or RFC 9220 claim and no primary performance ranking |
| Node `ws` | `org.protocol-lab.components.implementation.node-ws-websocket` | Thin canonical echo adapter, HTTP/1 WebSocket lane |
| Jetty WebSocket | `org.protocol-lab.components.implementation.jetty-websocket` | Thin canonical echo adapter with explicit HTTP binding variants supported by the existing contracts |
| uWebSockets | `org.protocol-lab.components.implementation.uwebsockets-websocket` | Later high-value native echo adapter after build and maintenance risks are bounded |

Apache RFC 8441 proxy behavior is not an independent WebSocket origin and must
not be registered as one.

## Program-level completion

The wishlist is complete when every named package above is either:

- implemented, built, admitted, and exercised by a compatible executor on the
  real ProtocolLab with an evidence bundle; or
- closed by a repository-backed feasibility/licensing decision showing that an
  unmodified upstream wrapper cannot satisfy the contract, with the exact gap
  retained as explicit follow-up rather than represented as support.

The final matrix must show package IDs, immutable versions and hashes, supported
and unsupported scenario rows, execution variants, controller jobs, validation
results, evidence locations, and comparison-cohort/ranking eligibility.

## Explicit non-goal

Do not add another general-purpose HTTP/3 server merely to increase the count.
Current HTTP/3 implementation breadth is already stronger than HTTP/2, TLS,
and secure DNS diversity; complete credible rows for existing HTTP/3 packages
separately from this program.
