# ProtocolLab Implementation Diversity Wishlist

Updated: 2026-07-14

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

## Reconciled delivery baselines

The implementation program uses these exact starting points so package and
runner work does not depend on dirty or divergent worktrees:

| Surface | Commit | Role | Baseline evidence |
| --- | --- | --- | --- |
| Public ProtocolLab contracts | `f8117f3967c35f91baa9a19277d52f9e2c4a0c85` | Exact HTTP/2 comparison topology plus the non-publishable `http2-performance-smoke` implementation selector | Suite schema validation passed; no implementation code |
| ProtocolLab components | `4fe33d6584354f8f31b1a3294abe49fa42b0fee1` | Consolidated scenario and executor package identities | Clean component integration head used by every implementation lane |
| ProtocolLab internal runner | `3aea501` (including `443e8e70e62e1670728411d70242206f0c2fe075`) | Consolidated exact executor admission and Protocol Execution Result v2 paths, package-backed Docker image materialization, and execution-only package capability application | All 34 focused `LabWorkerExecutorTests` and all 43 `LabPackageTests` passed. Controller bundle `bddb13cf...` and worker bundle `0c6d0b2a...` await the controller/SUT-worker refresh. |

Implementation work is integrated through
`codex/implementation-diversity-20260713`. Runner proof work is isolated in
`codex/implementation-diversity-runner-20260714`. Existing dirty worktrees and
the concurrent QUIC benchmark stream are not inputs to this program.

## Delivery ledger

Status values are `planned`, `implementing`, `local-proven`, `controller-admitted`,
`live-proven`, or `closed-by-decision`. A package is not `live-proven` until a
compatible controller job completes with validation and exact three-package
provenance.

| Cohort | Package(s) | Status | Local package evidence | Controller/job evidence |
| --- | --- | --- | --- | --- |
| HTTP/2 origins | `caddy-http2`, `nginx-http2` | controller-admitted | Extracted immutable packages `ceec7600...` and `74c60b2f...` passed both exact h2c scenarios with HTTP/2.0 observed and no fallback | Job `job-9bb04a815b6549e5af491bfd18f6a520` retained 164 artifacts but classified all cells `target-unsupported`: the deployed worker did not build absent package images. Runner fix `443e8e7` and worker bundle `0c6d0b2a...` are ready for refresh/retry. |
| HTTP origins | `apache-http1`, `apache-http2` | controller-admitted | Six-package Apache smoke passed exact HTTP/1.1 and h2c rows; current packages are HTTP/1 `0.1.2` / `57870a74...` and HTTP/2 `0.1.1` / `448ad643...` | Job `job-621e5d80db2e424f8eafc8bc78bad083` proved the HTTP/1 target healthy and both validations passed, but the stale deployed CLI omitted `PLAB_TARGET_BASE_URL`; benchmark evidence is not accepted until the refreshed-runner retry. |
| TLS endpoint tools | `openssl-s-server`, `gnutls-serv` | controller-admitted | Self-contained Docker packages `e270dcc3...` and `5bf19a9c...` passed exact `tls.handshake.full` TLS 1.3/AES-128-GCM/X25519/ALPN/certificate/non-resumption/zero-byte proof | Job `job-a290a600ef9b4037ad279a7ec6d01e47` retained 110 artifacts but classified both cells `validation-unsupported` because the deployed runner rejected the scenario's client role before starting the server. The current runner bundle is ready for refresh/retry. |
| TLS wrapper candidates | rustls example, s2n-tls utility | closed-by-decision | Repository-backed feasibility audit in `docs/tls-endpoint-tool-feasibility.md`; neither unmodified upstream utility exposes the complete package contract | Not applicable |
| Authoritative secure DNS | BIND DoT | controller-admitted | `bind9-dot@0.1.0` / `de8abf6e...` passed 5,241 exact local operations with TLS 1.3, ALPN `dot`, canonical answers, and zero failures/timeouts | Controller preview is 1 runnable, 0 unsupported/unavailable/invalid; live job awaits runner refresh. |
| Authoritative secure DNS | Technitium DoT/DoH2/DoH3/DoQ; BIND DoH2 | closed-by-decision | Exact ALPN, cipher, HTTP-header, redistribution, and current scenario/executor gaps are retained in package-local decision READMEs | Not applicable until the named gaps are closed. |
| Resolver secure DNS | Unbound DoT/DoH2 | closed-by-decision | Resolver-role semantics do not match the current authoritative scenario and ranking cohort; no false implementation package was created | Not applicable until a resolver scenario/cohort exists. |
| gRPC runtimes | grpc-dotnet, Java/Netty, C++, grpc-js | controller-admitted | Exact immutable executor smoke passed all 12 .NET, 9 Node, 2 Java, and 2 C++ declared rows with TLS 1.3, ALPN `h2`, and no fallback | Controller previews are 8 and 4 runnable cells with zero invalid cells; live jobs await runner refresh. |
| WebSocket runtimes | websocat, Node `ws`, Jetty, uWebSockets | controller-admitted | Extracted immutable Docker smoke passed 19/19 supported cells. Node, Jetty, and uWebSockets pass all five; websocat passes four and explicitly excludes binary echo. | Six packages are admitted/selectable. Controller preview is 19 runnable, 1 explicitly unsupported, 0 unavailable/invalid; live job awaits runner refresh. |

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
