# ProtocolLab Implementation Coverage Backfill Wishlist

Updated: 2026-07-15

This is the delivery backlog for the program after the initial implementation-
diversity milestone. That milestone proved that fifteen new implementation
packages could be admitted, selected immutably, exercised by the real
ProtocolLab controller, and represented on the public evidence site. This
program turns that proof into broad, useful implementation coverage.

The desired end state is not a large catalog with a few exercised rows. A
public implementation is either backed by current package and execution
evidence, or is clearly identified as a catalog-only candidate with the exact
reason it is not runnable. Every mature protocol lane has more than one
independent implementation, and the most important comparison lanes have a
small decision-ready evidence cohort.

## Program outcomes

- Backfill the existing QUIC and HTTP/3 catalogs instead of treating catalog
  visibility as exercised support.
- Give every secure-DNS transport at least two independent implementations in
  each role where the public contracts support comparison.
- Add recognizable independent implementations to HTTP/1.1, HTTP/2, TLS,
  gRPC, and WebSocket beyond the first-wave packages.
- Exercise WebTransport and MASQUE with at least two independent ecosystems
  once their existing public contracts and executors are runnable end to end.
- Produce real-lab artifacts for every supported target and explicit
  unsupported evidence for every declared-but-unavailable cell.
- Upgrade selected comparison cohorts from validation-only measurements to
  decision-ready evidence with repeated runs, controlled topology, complete
  provenance, variance, and saturation checks.
- Make the public site state the coverage denominator and evidence meaning in
  plain language.

## Non-negotiable rules

1. Protocol, transport, execution variant, and implementation role remain
   explicit. Origin, gateway, authoritative DNS, resolver, diagnostic tool,
   and fixture implementations never share a comparison cohort.
2. An implementation counts as live-proven only when an immutable package is
   admitted and materialized by the real controller and a compatible executor
   produces retained validation and measurement artifacts.
3. A custom adapter may provide only the application semantics that a protocol
   library requires. It must preserve the upstream protocol stack identity and
   implement the canonical fixture contract byte-for-byte.
4. Unsupported scenarios remain explicit. A nearby scenario, protocol
   fallback, proxy substitution, or different protobuf/DNS/application
   contract is not evidence for the requested cell.
5. Package versions and hashes are immutable. Evidence records the scenario,
   executor, implementation, toolchain, source/image provenance, raw output,
   topology, validation result, and normalized metrics.
6. Validation must pass before measurements are accepted. Decision-ready
   evidence additionally requires matched cohorts, isolated target and load
   roles, repeated runs, variance policy, and target/load saturation evidence.
7. A feasibility or licensing decision may explain why a candidate is not
   runnable, but it does not satisfy a protocol's minimum live-diversity floor.
8. Public pages must distinguish **cataloged**, **measured validation**,
   **comparable observation**, and **decision-ready benchmark** without
   requiring readers to understand internal evidence fields.

## Coverage floors

These floors are program completion requirements, not ceilings.

| Lane | Minimum live-proven breadth | Minimum decision-ready breadth |
| --- | ---: | ---: |
| Raw QUIC transport | 5 independent ecosystems | 3 on one common transport workload |
| HTTP/3 origins | 8 independent ecosystems | 4 on plaintext, JSON, and one fixed payload |
| HTTP/2 origins | 8 independent ecosystems | 4 per comparable h2c or TLS/ALPN cohort |
| HTTP/1.1 origins | 7 independent ecosystems | 4 on the shared origin workload |
| TLS endpoint/runtime | 6 independent ecosystems | 4 on TLS 1.3 full handshake |
| Authoritative DoT | 2 independent ecosystems | 2 |
| Authoritative DoH2 | 2 independent ecosystems | 2 |
| Authoritative DoH3 | 2 independent ecosystems | 2 comparable observations initially |
| Authoritative DoQ | 2 independent ecosystems | 2 comparable observations initially |
| Resolver DoT | 2 independent ecosystems | 2 |
| Resolver DoH2 | 2 independent ecosystems | 2 |
| gRPC | 5 independent runtimes | 4 on unary plus supported streaming rows |
| HTTP/1.1 WebSocket | 5 independent runtimes | 4 on the common echo/control workload |
| HTTP/2 WebSocket (RFC 8441) | 2 independent ecosystems | 2 comparable observations initially |
| HTTP/3 WebSocket (RFC 9220) | 2 independent ecosystems | 2 comparable observations initially |
| WebTransport over HTTP/3 | 2 independent ecosystems | 2 comparable observations initially |
| MASQUE CONNECT-UDP | 2 independent ecosystems | 2 comparable observations initially |

## Workstream A - secure DNS breadth

Authoritative and resolver roles use separate scenarios and comparison groups.
All local authoritative targets serve the deterministic `plab.test` zone with
no recursive upstream dependency.

### Authoritative targets

- [x] Keep BIND 9 DoT current and add BIND 9 DoH2 using the existing exact
  authoritative DNS fixture and no HTTP fallback.
- [ ] Add Technitium authoritative packages for DoT, DoH2, DoH3, and DoQ using
  externally acquired or source/config-only packaging if redistribution is not
  approved.
- [x] Promote the existing Go fixture targets for DoT, DoH2, DoH3, and DoQ to
  live evidence baselines, but label them as fixture implementations rather
  than production server products.
- [ ] Add classic UDP and TCP evidence for BIND 9 and Technitium so encrypted
  transport overhead can be studied without conflating roles.

### Resolver targets

- [ ] Add Unbound resolver packages for DoT and DoH2 with deterministic local
  upstream authority, cold/warm cache states, and DNSSEC validation state
  recorded.
- [ ] Add Knot Resolver packages for DoT and DoH2 under the same resolver
  contract and cache-state controls.
- [ ] Evaluate Knot Resolver and Technitium resolver modes for DoQ and DoH3;
  implement every lane that can satisfy the exact public contract and keep
  non-conforming lanes visible as unsupported.

### Secure-DNS evidence gate

- [ ] Run every supported secure-DNS package through its real executor and
  publish protocol-specific validation reports.
- [ ] Produce decision-ready DoT and DoH2 comparisons separately for
  authoritative and resolver roles; retain DoH3 and DoQ as comparable
  observations until topology and QUIC saturation gates are proven.

### Current retained secure-DNS evidence

| Role | Transport | Implementation | Immutable package version | Completed controller job |
| --- | --- | --- | --- | --- |
| Authoritative fixture | DoT | Go DNS DoT | 0.2.1 | `job-78dca1a735ed428c954e9b607ebb3c64` |
| Authoritative product | DoT | BIND 9 | 0.1.0 | `job-4568f06d29494058995f2a3bf7dab774` |
| Authoritative fixture | DoH2 | Go DNS DoH2 | 0.2.1 | `job-4bee9b68e59f4067bb5f292226a59b97` |
| Authoritative product | DoH2 | BIND 9 | 0.1.0 | `job-3e256dd1cb8a4f668464e2cd5e0eac7e` |
| Authoritative fixture | DoH3 | Go DNS DoH3 | 0.2.1 | `job-2f444bf1b8f34846806ed9eee34922c1` |
| Authoritative fixture | DoQ | Go DNS DoQ | 0.2.1 | `job-1c91832835d14fa8b049aa02ed8ccffd` |

These are real isolated-pair controller runs with retained raw and normalized
artifacts. They establish current package proof and the two-implementation DoT
and DoH2 breadth counts. They do not yet satisfy the repeated-run,
variance/saturation, publication, resolver-role, DoH3/DoQ second-ecosystem, or
decision-ready gates below.

## Workstream B - raw QUIC transport backfill

- [ ] Keep `quic-go-raw` current across handshake/echo, 1 MiB stream
  throughput, multiplexing, and duplex workloads.
- [ ] Package and live-prove the Incursa `quic-dotnet` raw target through its
  implementation-owned handoff without coupling the runner to Incursa code.
- [ ] Package and live-prove the existing MSQuic/.NET raw adapter with explicit
  platform capability metadata.
- [ ] Add raw transport packages for picoquic, Quinn, and s2n-quic.
- [ ] Add compatibility-first raw packages for aioquic, quiche, ngtcp2, XQUIC,
  LSQUIC, neqo, and mvfst where the upstream interop image exposes an exact
  scenario mapping.
- [ ] Record client/server role and supported QUIC interop testcases for every
  package; do not treat an HTTP/3 origin as a raw QUIC target.
- [ ] Live-run all supported raw packages and meet the raw QUIC coverage floor
  with a three-ecosystem decision-ready common cohort.

## Workstream C - HTTP/3 catalog backfill

### Existing package completion

- [ ] Bring Kestrel, Incursa HTTP/3, Caddy, nginx, quic-go, and aioquic onto a
  common plaintext, JSON, 1 KiB, and 64 KiB support matrix where their exact
  semantics permit it.
- [x] Re-run the current immutable quic-go and aioquic package heads so live
  evidence matches the cataloged package versions.
- [ ] Complete diagnostic peer-characterization evidence for quiche and
  ngtcp2, then add official payload rows only where status, content type,
  length, and payload bytes satisfy the canonical workload.
- [ ] Exercise h3spec/QPACK against every compatible HTTP/3 target and retain
  exact unsupported or failed requirements.

### Current retained HTTP/3 evidence

| Implementation | Immutable package version | Proven rows | Completed controller job |
| --- | --- | --- | --- |
| aioquic | 0.3.2 | canonical JSON status and 1 KiB payload; validation and measurement passed | `job-0d08b2ace1704d609ec9803e6e7119c7` |
| aioquic | 0.3.3 | h3spec status, response-header, and QPACK diagnostics; all 15 requests succeeded in each cell | `job-a3c8b35637e14c49b86332a928c5b15d` |
| quic-go | 0.1.6 | canonical JSON status, 1 KiB, and 64 KiB payloads; validation and measurement passed | `job-610e9f2d38364cfc95b238ea6e012446` |
| Kestrel | 0.1.6 | canonical JSON status, 1 KiB, and 64 KiB payloads; validation and measurement passed | `job-fb08e6a527b94ee1a922055a9401feee` |

These runs used the package-backed managed HTTP/3 executor and retained
the executor package identity, requested/effective load shapes, raw output,
target-container telemetry, normalized metrics, and immutable target package
provenance. They close current-head proof only for the rows shown. The current
aioquic head now has h3spec/QPACK proof, while its 64 KiB payload row, the
remaining compatible-target h3spec/QPACK matrix, repeated comparison,
publication, and decision-ready gates remain.

### New catalog packages

- [ ] Package and live-prove XQUIC, LSQUIC, and neqo from reproducibly pinned
  upstream/interop artifacts.
- [ ] Package mvfst/Proxygen as an HTTP/3 origin only if the Proxygen layer
  satisfies the origin contract; keep raw mvfst evidence separate.
- [ ] Add H2O as an experimental HTTP/3 origin with that status visible in
  metadata and public presentation.
- [ ] Add HAProxy as a gateway/proxy cohort, never as an origin-server row.
- [x] Reconcile every HTTP/3 public catalog entry to one of: current
  live-proven package, current catalog-only candidate with exact blocker, or
  role-correct removal from the HTTP/3 origin cohort.
- [ ] Meet the HTTP/3 coverage floor and produce a four-origin decision-ready
  common cohort.

## Workstream D - HTTP/1.1 and HTTP/2 origin breadth

- [ ] Keep Kestrel, Caddy, nginx, and Apache packages aligned across their
  exact supported HTTP/1.1 and HTTP/2 origin semantics.
- [ ] Add Go `net/http` HTTP/1.1 and HTTP/2 origin packages.
- [ ] Add Node.js `node:http` and `node:http2` origin packages.
- [ ] Add Rust hyper HTTP/1.1 and HTTP/2 origin packages.
- [ ] Add Jetty HTTP/1.1 and HTTP/2 origin packages.
- [ ] Preserve separate HTTP/2 h2c-prior-knowledge and TLS/ALPN execution
  variants; neither mode may provide evidence for the other.
- [ ] Keep gateways/proxies in a separate cohort and add HAProxy HTTP/1.1 and
  HTTP/2 gateway packages only after that cohort is represented publicly.
- [ ] Meet both HTTP origin coverage floors and publish decision-ready common
  cohorts without implying cross-protocol rankings.

## Workstream E - TLS implementation breadth

- [ ] Retain OpenSSL `s_server` and GnuTLS `gnutls-serv` as diagnostic endpoint
  tools with their exact control limitations visible.
- [ ] Promote .NET `SslStream`, Go `crypto/tls`, rustls, and s2n-tls into a
  comparable TLS 1.3 full-handshake cohort using minimal protocol-library
  adapters where upstream utilities cannot host the canonical fixture.
- [ ] Add a wolfSSL endpoint/runtime package if its license and reproducible
  build satisfy the package rules.
- [ ] Keep TLS 1.2, mTLS, resumption, early data, cipher-specific, record, and
  KeyUpdate rows as separate capability cohorts rather than shrinking the
  common TLS 1.3 denominator.
- [ ] Live-run all supported TLS packages, meet the breadth floor, and produce
  a four-runtime decision-ready TLS 1.3 full-handshake cohort.

## Workstream F - gRPC and WebSocket breadth

### gRPC

- [ ] Live-prove the existing Go gRPC/H2 target and add it to the public gRPC
  implementation cohort.
- [ ] Bring grpc-dotnet, grpc-go, grpc-java/Netty, grpc-cpp, and grpc-js onto a
  common unary/server-streaming/client-streaming/bidirectional matrix with
  terminal deadline and cancellation rows where supported.
- [ ] Produce a four-runtime decision-ready unary cohort and comparable
  observations for streaming and terminal behavior.

### WebSocket

- [ ] Live-prove the existing Go HTTP/1.1 WebSocket target alongside Node
  `ws`, Jetty, uWebSockets, and websocat, preserving websocat's diagnostic-only
  limitations.
- [ ] Add a second RFC 8441 implementation alongside Kestrel HTTP/2 WebSocket.
- [ ] Add a second RFC 9220 implementation alongside aioquic HTTP/3 WebSocket,
  preferring ngtcp2/nghttp3 or another cataloged stack with explicit support.
- [ ] Keep cleartext, TLS, RFC 8441, and RFC 9220 as separate transport cohorts
  and meet each applicable breadth/evidence floor.

## Workstream G - WebTransport and MASQUE

- [x] Audit the existing WebTransport and MASQUE public contracts against the
  current component executor and scenario surfaces before adding packages.
- [ ] Package webtransport-go plus one independent WebTransport ecosystem and
  live-prove the common session/stream/datagram contract.
- [ ] Package two independent MASQUE CONNECT-UDP implementations with explicit
  proxy and target roles and no ordinary HTTP proxy substitution.
- [ ] Publish comparable observations for both protocols; decision-ready
  ranking is deferred until the public comparison policy defines meaningful
  cohort and topology controls.

## Workstream H - evidence and public explanation

- [ ] For every named live package, retain a real-controller job ID, immutable
  package identity/version/hash, worker capability match, validation result,
  raw executor output, normalized measurements, and evidence bundle location.
- [ ] Run at least three clean repetitions for comparable observations and at
  least seven for decision-ready candidates unless the public evidence policy
  adopts a stricter requirement.
- [ ] Require physically separated target/load roles, controlled network/CPU
  conditions, source/image parity, variance within policy, and non-saturation
  evidence before a result is decision-ready.
- [ ] Publish accepted reports through the normal report-import pipeline and
  verify their protocol, implementation, workload, run, artifact, and
  comparison pages on `lab.incursa.com`.
- [x] Replace ambiguous top-level public labels with the four-state vocabulary
  in this document while preserving technical claim level, publishability,
  evidence class, topology, validation, and exclusion details in disclosures.
- [ ] Show tested/admitted/cataloged denominators on every protocol hub and an
  explicit reason for every catalog-only or non-ranked implementation.

## Delivery order

1. Reconcile machine-readable coverage and exact blockers for every current
   catalog entry.
2. Secure DNS authoritative breadth, beginning with the second DoT target and
   real DoH2/DoH3/DoQ server packages.
3. Existing HTTP/3 package-head reruns and raw QUIC first-party/MSQuic handoff.
4. New raw QUIC and HTTP/3 catalog packages seeded from maintained upstream or
   interop artifacts.
5. HTTP/1.1, HTTP/2, TLS, gRPC, and WebSocket breadth additions.
6. Resolver secure DNS, WebTransport, and MASQUE role-specific cohorts.
7. Controlled decision-ready campaigns and public terminology/coverage update.

Independent package work may proceed in parallel after contracts and shared
executor behavior are stable. Live campaigns sharing workers or network
resources run serially to avoid contaminating evidence.

## Program completion

This wishlist is complete only when all of the following are true:

1. Every checkbox above is satisfied or replaced by a dated, repository-backed
   blocker decision; blocker decisions do not reduce any coverage floor.
2. Every coverage floor is met by current immutable packages with real-lab
   evidence, not just manifests, local smoke, or historical versions.
3. Every supported cell has retained validation and measurement artifacts, and
   every unsupported cell has a precise visible reason.
4. Each mature lane has the required decision-ready cohort; emerging lanes
   explicitly limited to comparable observations are labeled as such.
5. The public site exposes the full implementation and evidence matrix and a
   reader can tell what was cataloged, measured, comparable, and decision-ready
   without reading internal documentation.
6. All changed repositories pass their repository-specific validation, are
   merged to `main`, pushed, deployed through their approved publication path,
   and end with clean worktrees while preserving unrelated user work.
