# ProtocolLab Implementation Coverage Backfill Wishlist

Updated: 2026-07-16

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
- [x] Evaluate source/config-only Technitium authoritative wrappers for DoT,
  DoH2, DoH3, and DoQ. Package-local decision records close all four encrypted
  lanes against the current exact contracts: DoT lacks required ALPN and cipher
  control, DoH2 lacks `Cache-Control: no-store`, DoH3 has that header mismatch
  plus uncontrollable QUIC cipher selection, and DoQ lacks exact cipher control.
- [x] Promote the existing Go fixture targets for DoT, DoH2, DoH3, and DoQ to
  live evidence baselines, but label them as fixture implementations rather
  than production server products.
- [x] Add classic UDP and TCP evidence for BIND 9 so encrypted transport
  overhead can be studied without conflating roles.
- [x] Add classic UDP and TCP evidence for Technitium under the same exact
  authoritative role and deterministic zone contract.

### Resolver targets

- [x] Add a public resolver-specific DoT scenario, deterministic fixture,
  suite, and cache-control artifact contract without mixing resolver rows with
  authoritative rows.
- [x] Add a native BIND 9 recursive DoT package with an isolated local fixture
  authority and authenticated per-operation cache flush. Package-local exact
  wire, TLS, ALPN, role, and upstream-isolation proof passes; real-lab proof is
  tracked by the evidence gate below.
- [x] Evaluate Unbound resolver packages for DoT and DoH2 with deterministic
  local upstream authority, cache-state control, and DNSSEC validation state
  recorded. Native Unbound DoH2 is implemented; native Unbound DoT is closed
  against the current contract because it does not negotiate the required
  `dot` ALPN, and a TLS terminator would substitute the measured transport.
- [x] Add Knot Resolver packages for DoT and DoH2 under the same resolver
  contract and cache-state controls.
- [x] Evaluate Knot Resolver and Technitium resolver modes for DoQ and DoH3;
  implement every lane that can satisfy the exact public contract and keep
  non-conforming lanes visible as unsupported. The dated
  [resolver QUIC feasibility decision](secure-dns-resolver-quic-feasibility-2026-07-16.md)
  records why none can currently declare an exact resolver-role DoQ or DoH3
  cell without role substitution or transport-policy overclaiming.

### Secure-DNS evidence gate

- [x] Run every currently supported secure-DNS package through its real executor and
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
| Authoritative product | Classic DNS/UDP | BIND 9 | 0.1.0 | `job-88d1dbab8b6a40b7b43992995a02b219` |
| Authoritative product | Classic DNS/TCP | BIND 9 | 0.1.0 | `job-799e5c5f78f44fb4805a23a2ff6a8744` |
| Authoritative product | Classic DNS/UDP | Technitium DNS Server | 0.1.1 | `job-080eac8e1a4542fe9853023404077792` |
| Authoritative product | Classic DNS/TCP | Technitium DNS Server | 0.1.1 | `job-0ada77e3fbad47928dec7cc74b3150ef` |
| Recursive resolver | DoT | BIND 9 | 0.1.2 | `job-b24f3e383cca482483e8ec6115147afd` |
| Recursive resolver | DoH2 | Unbound | 0.1.2 | `job-211960ceb67d47ff8b9becb6da92f533` |
| Recursive resolver | DoT | Knot Resolver | 0.1.5 | `job-2312532e03364be7b28802d5743531da` |
| Recursive resolver | DoH2 | Knot Resolver | 0.1.5 | `job-359859224d4a4bf2be9dd427a21883ac` |

These are real controller runs with retained raw and normalized
artifacts. They establish current package proof and the two-implementation DoT
and DoH2 breadth counts. The BIND and Technitium classic rows are published
diagnostic observations with accepted validation and measurements; they are
not ranked or decision-ready. The BIND, Unbound, and Knot Resolver rows additionally
retain cache-flush and local-only upstream proof and are visible on
`lab.incursa.com` as measured validation. They remain diagnostic/unranked
because they use one repetition and shared target/load worker placement. The
secure-DNS set now satisfies the two-resolver live breadth floor for DoT and
DoH2, but does not yet satisfy the repeated-run, variance/saturation,
DoH3/DoQ second-ecosystem, or decision-ready gates below.

## Workstream B - raw QUIC transport backfill

- [x] Keep `quic-go-raw` current across handshake/echo, 1 MiB stream
  throughput, multiplexing, and duplex workloads.
- [x] Package and live-prove the Incursa `quic-dotnet` raw target through its
  implementation-owned handoff without coupling the runner to Incursa code.
- [x] Package and live-prove the existing MSQuic/.NET raw adapter with explicit
  platform capability metadata.
- [x] Add raw transport packages for picoquic, Quinn, and s2n-quic.
  - [x] picoquic
  - [x] Quinn
  - [x] s2n-quic
- [ ] Add compatibility-first raw packages for aioquic, quiche, ngtcp2, XQUIC,
  LSQUIC, neqo, and mvfst where the upstream interop image exposes an exact
  scenario mapping.
  - [x] aioquic
  - [x] quiche
  - [ ] ngtcp2
  - [ ] XQUIC
  - [ ] LSQUIC
  - [ ] neqo
  - [ ] mvfst
- [ ] Record client/server role and supported QUIC interop testcases for every
  package; do not treat an HTTP/3 origin as a raw QUIC target.
- [ ] Live-run all supported raw packages and meet the raw QUIC coverage floor
  with a three-ecosystem decision-ready common cohort.

### Current retained raw QUIC evidence

| Implementation | Immutable package version | Proven rows | Completed controller job |
| --- | --- | --- | --- |
| Incursa `quic-dotnet` | `dev-20260716T032138Z-c4c53766-clean` | 1 MiB stream throughput, 100x64 KiB multiplexing, and duplex streams; five accepted repetitions each | `job-ceced0c711554077b48af4195996efd4` |
| MSQuic/.NET | `dev-20260716T034228Z-8a408704-clean` | 1 MiB stream throughput, 100x64 KiB multiplexing, and duplex streams; five accepted repetitions each | `job-73d3893c6b70459386e6fbf2428478f0` |
| quic-go | `0.1.16` | Cold handshake, 1 KiB echo, 1 MiB stream throughput, 100x64 KiB multiplexing, and 16-stream duplex; current-head rows have accepted cross-worker measurements | `job-dfac45aec77143ef89781b85d50b5419`, `job-618030b559764ac98686ea78cd458257`, `job-9d4c7d6fa1c74ceea41aea0f91a2b84f`, `job-95c2770be54245388ff236cf2d508379`, `job-1df61cda4267436f98dd9ac7612266c4` |
| Quinn | `0.1.0` | Cold handshake, 1 KiB echo, 1 MiB stream throughput, 100x64 KiB multiplexing, and duplex streams; every row has one accepted cross-worker measurement | `job-d81806366b4e447dab866e5f9652cbef`, `job-5eabc52f893046b7afc23fc67ba138b7`, `job-2bb41e44065f4c048289f2cfa1f20ad9`, `job-53da1c1d431e454ea4908a1809cc94f3`, `job-5c85e53f5dfe493ca30196b2a1a0fec3` |
| s2n-quic | `0.1.0` | Cold handshake, 1 KiB echo, 1 MiB stream throughput, 100x64 KiB multiplexing, and duplex streams; every row has one accepted cross-worker measurement | `job-2e586cdfd6084ece82f91d327d9f95c7`, `job-d507de39055e4b01b53c5f0e4cb0e4b4`, `job-00fbbb776aa94930b1bfcab074af4119`, `job-555ac25d829746baa9722cc6d2bdc3f6`, `job-4c3d644cd576452da12e586d1b64bfb6` |
| picoquic | `0.1.0` | Cold handshake, 1 KiB echo, 1 MiB stream throughput, 100x64 KiB multiplexing, and duplex streams; every row has one accepted cross-worker measurement | `job-3150078923e34eac9a1c957be289de91`, `job-4cb1795c45ae4ebe89a528cc2b9cda98`, `job-8a40e17c3b164247a35209add5ed212c`, `job-207ec5092cd74380a130441eb51d0236`, `job-bdefa794c67541648a5ab7d4026aeb1e` |
| aioquic | `0.1.0` | Cold handshake, 1 KiB echo, 1 MiB stream throughput, 100x64 KiB multiplexing, and duplex streams; every row has one accepted cross-worker measurement | `job-12341c52aa404a19b1e8ccb9c8a71e80`, `job-6c1e3f6aee044cc7b967819995d2d0c7`, `job-9218d24df059426892d50f620482a782`, `job-0e91044175c34a0999f5e02263c32cfa`, `job-e79487caff9d44889973184d7cb4a3fa` |
| quiche | `0.1.0` | Cold handshake, 1 KiB echo, 1 MiB stream throughput, 100x64 KiB multiplexing, and duplex streams; every row has one accepted cross-worker measurement | `job-dab2f6a8587d44568fe0cbe93e99621e`, `job-fa095524f84e4cfa861d5c2aa6dbd350`, `job-7a9d939a10d34fcf88b0acec9f704d6d`, `job-896ab3b02b27452ebcce08ca10b42d5a`, `job-49d9885474d34a0a8e5d292b6501141d` |

These clean implementation-owned packages were selected with immutable executor
and scenario-package hashes. Each public report retains all 15 accepted cells
and their raw and normalized artifacts. They are live-proven diagnostic
evidence, not decision-ready evidence: target and load execution still shared
one worker, and both reports correctly remain `publishable=false`.

The current quic-go target and executor heads are immutable package
`org.protocol-lab.components.implementation.quic-go-raw@0.1.16`
(`8d18e7bd2aec...`) and
`org.protocol-lab.components.executor.quic-go-raw-load@0.1.13`
(`789de256d5b0...`). The cross-worker proof used
`plab-worker-load-01 -> plab-worker-sut-01` and is published as
[`raw-quic-peer-matrix-raw-quic-peer-handshake-cold-raw-quic-transport-v1-smoke-cell-1`](https://lab.incursa.com/reports/raw-quic-peer-matrix-raw-quic-peer-handshake-cold-raw-quic-transport-v1-smoke-cell-1).
It retains one accepted measurement and the exact topology but remains
diagnostic/unranked because both VMs share physical host `r920` and only one
repetition was run. The first isolated attempt
(`job-9d0fb4f7ac504d07bdd3cae374a52f16`) is retained as failed ALPN evidence: the
SUT role selected an unsupported catalog cell. Worker commit `eab14bf` now
selects the single runnable resolved cell and rejects ambiguous multi-cell
isolated jobs. A subsequent deployment-only permission failure
(`job-84dcc68aa98a4f9dbb0979dd07613f49`) is also retained and was closed by
restoring the `protocol-lab:protocol-lab` bundle ownership contract. Echo,
1 MiB throughput, 100x64 KiB multiplexing, and 16-stream duplex then completed
with one accepted cross-worker measurement and 94 retained controller
artifacts each. All five reports were verified live on `lab.incursa.com`.
The rejected generic-smoke multiplex attempt
(`job-28cb5e982e5c4ba58e70aec9c0a8aef8`) is retained too: the runner correctly
refused to replace the named 100-stream workload with the profile's one-stream
shape. The completed rerun uses `raw-quic-peer-confidence` and records
`c1-s100-r1`. This closes current-head quic-go breadth, not the repeated,
physically isolated decision-ready gate.

The Quinn target is immutable package
`org.protocol-lab.components.implementation.quinn-raw@0.1.0`
(`35ca2c4ba421...`) built from clean component commit `fd7e122`. It uses Quinn
`0.11.11`, the exact `plab-raw-quic` ALPN, and the same pinned executor and
scenario packages as the current quic-go proof. All five runs used the real
isolated-pair path `plab-worker-load-01 -> plab-worker-sut-01`; handshake
retained 102 controller artifacts and the other four retained 94 each. Their
publication attempts (`pub_db0cf757995244dc88e559b3c88d232c`,
`pub_01b7083ba6474c4a927221445743c764`,
`pub_7a708c1bec98467ea3f1e03c9d25c7a1`,
`pub_26e740b6b4f645f49e5fbdf36621774d`, and
`pub_a237869114494fc9a97df0deec65e12b`) uploaded and verified the public
objects and were accepted by the site import queue. All five report routes and
the composite `/implementations/quinn-raw` drill-down were verified live. The
measurements remain diagnostic/unranked because they use one repetition and
the target/load VMs share physical host `r920`; Quinn adds a fourth live raw
QUIC ecosystem but does not close the five-ecosystem or decision-ready floor.

The s2n-quic target is immutable package
`org.protocol-lab.components.implementation.s2n-quic-raw@0.1.0`
(`a2276469e011...`) built from clean component commit `bd05c6e`. It uses
s2n-quic `1.83.0` with its rustls provider, the exact `plab-raw-quic` ALPN,
and the same pinned executor and scenario packages. All five runs used
`plab-worker-load-01 -> plab-worker-sut-01`; handshake retained 102 controller
artifacts and the other four retained 94 each. Publication attempts
`pub_f915de5d9bd741a283bfdcb66e195fec`,
`pub_f69f978e99684ff0934cd7300c341267`,
`pub_f25cda9c77f844678581a5225ed4119e`,
`pub_d995b0cc0e914a0ebc9cdbff01d4cf18`, and
`pub_d2efbe82dd2b4b86bdb2132c105fd287` all uploaded and verified their public
objects and were accepted by the site import queue. All five report routes and
the composite `/implementations/s2n-quic-raw` drill-down were verified live.
This reaches the five-ecosystem raw QUIC breadth floor with three common
workloads, but the rows remain diagnostic/unranked because they use one
repetition and the target/load VMs share physical host `r920`; the repeated,
physically isolated decision-ready cohort is still open.

The picoquic target is immutable package
`org.protocol-lab.components.implementation.picoquic-raw@0.1.0`
(`8e55c4749732...`) built from clean component commit `830e9c2`. It pins
picoquic commit `13671ce7bdf58c278a29da2d49a32f76c21d6c6d` and picotls commit
`bfa67875982afc4c24f21e146cef4747fa189c2f`, negotiates the exact
`plab-raw-quic` ALPN, and preserves the same canonical stream wire behavior as
the common raw cohort. All five jobs used `plab-worker-load-01 ->
plab-worker-sut-01`; handshake retained 92 controller artifacts and the other
four retained 83 each. Publication attempts
(`pub_dc34593bcfa440d48330b883dafc358f`,
`pub_04434134e2954c58a4a8e4e9ef9a2c96`,
`pub_942fb811cfe64f17a5f481ea34afffd1`,
`pub_52b1c01186d849f0ab80a501d2344c36`, and
`pub_caac479c0297466eb96f92f221aabd79`) uploaded and verified the public
objects and were accepted by the site import queue. All five report routes and
the composite `/implementations/picoquic-raw` drill-down were verified live.
picoquic adds another fully common raw implementation but does not upgrade the
cohort beyond diagnostic/unranked evidence: it still uses one repetition and
the target/load VMs share physical host `r920`.

The aioquic target is immutable package
`org.protocol-lab.components.implementation.aioquic-raw@0.1.0`
(`3533f1b7d81e...`) built from clean component commit `8bedf9b`. It bundles
aioquic `1.3.0` into a Linux x64 executable, negotiates the exact
`plab-raw-quic` ALPN, and implements only the canonical stream behaviors
declared by the package. All five jobs used `plab-worker-load-01 ->
plab-worker-sut-01`; handshake retained 102 controller artifacts and the other
four retained 94 each. Publication attempts
(`pub_9f0b51bbbf5e4c9ebc4f49f8b4366ce7`,
`pub_32f9564795764925a6ff31124025e9f0`,
`pub_c5c41fee4bca41088de85f5ee43f8a74`,
`pub_5f0c0261c96d41ccac326babdfddc719`, and
`pub_d464ca2661864f63ab5248bb9e7d356f`) uploaded and verified the public
objects and were accepted by the site import queue. All five report routes
were verified live. The rows remain diagnostic/unranked because they use one
repetition and the target/load VMs share physical host `r920`.

The quiche target is immutable package
`org.protocol-lab.components.implementation.quiche-raw@0.1.0`
(`7fa952c88af1...`) built from clean component commit `2ca3eb3`. It uses
Cloudflare quiche `0.29.3`, negotiates the exact `plab-raw-quic` ALPN, and
keeps the UDP event loop and canonical application stream behavior separate
from the existing quiche HTTP/3 peer package. All five jobs used
`plab-worker-load-01 -> plab-worker-sut-01`; handshake retained 102 controller
artifacts and the other four retained 94 each. Publication attempts
(`pub_62bf69e7a67e430cae7e149838f6ba08`,
`pub_538490da6e4c4fee9bca16fc059bb653`,
`pub_4e816db3d7814b3784ff77fc11486682`,
`pub_2b94b07f92294bc3aefb1f3c4c4c80c0`, and
`pub_df078fe77d4c43c4b7942c31dd3ed810`) uploaded and verified the public
objects and were accepted by the site import queue. All five report routes
were verified live. The rows remain diagnostic/unranked because they use one
repetition and the target/load VMs share physical host `r920`.

## Workstream C - HTTP/3 catalog backfill

### Existing package completion

- [ ] Bring Kestrel, Incursa HTTP/3, Caddy, nginx, quic-go, and aioquic onto a
  common plaintext, JSON, 1 KiB, and 64 KiB support matrix where their exact
  semantics permit it.
- [x] Re-run the current immutable quic-go and aioquic package heads so live
  evidence matches the cataloged package versions. aioquic `0.3.3` and
  quic-go `0.1.8` now have current package-backed h3spec/QPACK proof.
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
| Caddy | 0.1.9 | h3spec status, exact 50x32 response-header fixture, and QPACK diagnostics; validation and benchmark succeeded | `job-e05ec641965748f3830b5a006b3d8425` |
| nginx | 0.1.9 | h3spec status, exact 50x32 response-header fixture, and QPACK diagnostics; validation and benchmark succeeded | `job-e05ec641965748f3830b5a006b3d8425` |
| quic-go | 0.1.8 | h3spec status, exact 50x32 response-header fixture, and QPACK diagnostics; validation and benchmark succeeded | `job-e05ec641965748f3830b5a006b3d8425` |
| Kestrel | 0.1.8 | h3spec status, exact 50x32 response-header fixture, and QPACK diagnostics; validation and benchmark succeeded | `job-6c8c02669dbf423fbcc7a12a5d364741` |

These runs used the package-backed managed HTTP/3 executor and retained
the executor package identity, requested/effective load shapes, raw output,
target-container telemetry, normalized metrics, and immutable target package
provenance. They close current-head proof only for the rows shown. The current
aioquic, Caddy, nginx, quic-go, and Kestrel heads now have published
h3spec/QPACK proof. Their report pages are visible at `lab.incursa.com` with
package provenance and retained artifact links. aioquic 64 KiB, current-head
payload reruns for the newly bumped packages, Incursa HTTP/3, explicit
quiche/ngtcp2 compatibility outcomes, repeated comparison, and decision-ready
gates remain.

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
- [x] Add Go `net/http` HTTP/1.1 and HTTP/2 origin packages.
- [x] Add Node.js `node:http` and `node:http2` origin packages.
- [x] Add Rust hyper HTTP/1.1 and HTTP/2 origin packages.
- [x] Add Jetty HTTP/1.1 and HTTP/2 origin packages.
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

- [x] Live-prove the existing Go HTTP/1.1 WebSocket target alongside Node
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
