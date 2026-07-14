# ProtocolLab HTTP Executor Delivery Plan

Status: active engineering plan
Observed: 2026-07-13
Owner: `protocol-lab-components`, with runner integration owned by `protocol-lab-internal`

This plan turns the approved public HTTP performance contracts into package-backed test-executor lanes. It begins with HTTP/1.1 and HTTP/2, records the existing HTTP/3 gaps without refactoring that lane, and sequences TLS, gRPC over HTTP/2, secure DNS, and WebSocket work only after the HTTP executor interface is proven.

The public `protocol-lab` repository remains the workload and evidence-contract authority. This document does not create scenario semantics, authorize publication, or imply benchmark readiness. Public authority for the activated TLS, gRPC/H2, and secure-DNS v2 contracts is committed at `8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574`; component scenario packages lock exact authority files and hashes to that commit.

## 2026-07-13 vertical completion record

The local diagnostic implementation began with the four verticals in the table below. The consolidated checkpoint following the table records the later identity breadth. Every accepted cell materialized one implementation package, one test-executor package, and one scenario package; passed fail-closed parser/admission; preserved executor stdout/stderr and protocol-specific raw artifacts; and emitted a schema-valid, artifact-hash-valid Protocol Execution Result v2. These are same-host local smoke results, not comparison, ranking, or publishable evidence.

| Lane | Exact selection | Completed / failed / timed out | Final Windows package SHA-256 values | Final evidence root |
| --- | --- | --- | --- | --- |
| TLS 1.3 | `tls.handshake.full`; `tls-performance-smoke`; `tls-smoke` | `1552 / 0 / 0` | scenario `aedf58511b7631642787610e2bcbfdd30dbcacf72e8621c17367cd209c360631`; executor `1c19c9b35e3b44e2fd3686022cccd89e35d6e3f2db96512cef2a5aba7614e412`; target `1659e969be5026496c216981290206efa57ba6ed78f90cb632dec5f36b76e119` | `protocol-lab-http12-runner-bridge-clean/artifacts/final-tls-evidence/tls13-final-evidence-direct-package-cell` |
| gRPC/H2 | `grpc.h2.unary.echo`; `grpc-h2-performance-smoke`; `grpc-h2-smoke` | `1 / 0 / 0` | scenario `a8e6c7409d249157763942edc057f19a11cd27628bd002611dbc4709dc2885be`; executor `1e17206aae646916af702d55157bcd7de5716209cbc69a9dbfba18fbc8f1e78e`; target `45c0673a1de59881cbed1e04926c60b630bde9c121ed170fb766400c90551ac7` | `protocol-lab-http12-runner-bridge-clean/artifacts/final-runner-evidence/grpc-h2-evidence-direct-package-cell` |
| DNS over TLS | `dns.dot.query.a`; `dns-dot-performance-smoke`; `secure-dns-smoke` | `75679 / 0 / 0`; malformed `0`; retried `0` | scenario `b57f1c2355dac681ec2d317b121022dfb36fd22f5e15fe3e8adc1bec69920ba6`; executor `875ed46aff400a68c124923323346284d8298f0ab21b033d7019a93f92454883`; target `27b8dd0e8b674642953c6d75c69b376fa0b7c66a7e88d13e3e5c56b8d8fa37b6` | `protocol-lab-http12-runner-bridge-clean/artifacts/final-runner-evidence/dns-dot-evidence-direct-package-cell` |
| DNS over HTTPS/2 | `dns.doh2.query.a`; `dns-doh2-performance-smoke`; `secure-dns-smoke` | `16417 / 0 / 0`; malformed `0`; retried `0` | scenario `fa213ce602233de3843e1950bbfbef47b55e4fc7b8544deba964343845f214cf`; executor `1ef90b00315fa15e124ea3e75c3bc084bdee0047821b635e48b470ba1422706c`; target `11f9fcd4924ec7c33f97ee53cb92ee9557aeecdc5ad4558f2c702acb77e587f8` | `C:/shared/src/incursa/.artifacts/doh2-runner-evidence/doh2-final-runner-v3-direct-package-cell` |

### Consolidated executor-identity checkpoint

The clean component package head is `9b37446`, with TLS evidence follow-ups through `bf8b7ec`; the runner integration head is `5bad8ff`, including the integration-only TLS record load-shape repair, exact DoQ admission, HTTP/1.1 TLS WebSocket diagnostic admission, exact mutual-TLS admission, exact TLS 1.2 admission, and exact ChaCha20 admission. The consolidated runner and CLI build with zero warnings and errors. The latest widened TLS runner gate passes `256/256`, package/manifest regressions pass `51/51`, and the preceding combined TLS, gRPC, secure-DNS, and WebSocket selection gate passed `274/274`. The component integration validates `62` public and `62` internal manifests; focused race tests and `go vet` pass for every consolidated Go executor and target.

- TLS now has exact runner-backed support for `tls.handshake.full`, `tls.handshake.resumed`, `tls.record.throughput`, and `tls.record.coverage`. The record cells completed `691 / 0 / 0` and one six-case operation with `0 / 0`; their Windows package hashes are scenario `acbdd31687d388f9e236016475fc754d7f5ad5d0b43b3d5cb92443d61bde58c1`, executor `072a193facfc56d9e6620822205ab287e820ef0bb595b757492cd38255691e91`, and target `8140204a8db076c5af68c16f80686683aa6a7cbb0b009c55efb1f9e8bfb86ec3`. Record coverage remains experimental and requires explicit operator opt-in.
- TLS 1.2 compatibility has exact package and runner support for `tls.handshake.full.tls12`. Package hashes are scenario `2302d553231d4e5b0f134ce68113146690b0883acf899fa1db6286517852f655`, executor Windows `d7c404f489c74907294636a074ba79cd3db8458eaf97e6d4e4435c6ee3af7596` and Linux `b32108d753299104a0c249511158367d6f5492f62ad84f2ad4bf86397fe1044e`, and target Windows `c2edceccb04efc3e9710db038eb12c417def99dfd012adfb5c95309ba8249f7b` and Linux `2b452abe207617d84db0026d69939dfe9a725d04354d8725dd491741c7504a7f`. The real extracted-package runner cell `C:\shared\src\incursa\.artifacts\tls12-runner-evidence\tls12-final-v1-direct-package-cell` completed `3,067 / 0 / 0` with zero application bytes and exact TLS 1.2, ECDHE-ECDSA AES-128-GCM, X25519, ALPN, certificate, fresh-session, and no-early-data proof. The public Protocol Execution Result v2 schema passes and all `9/9` referenced artifact hashes verify.
- TLS mutual authentication has exact package and runner support for `tls.handshake.mutual-auth`. Package hashes are scenario `98708702ccae982c289729c0117cc4dfa64aa144c576088d8314d3e48aecb9d2`, executor Windows `109c9f381b6cf2a0ba11f9aaf5175146be58326f74dde4f8361f2bbed423ab3b` and Linux `08e5304f43415bb4ba1c03c95358a3bed58ecd1634b6d11fb0de37225498f63f`, and target Windows `6b5f348f676db0c163949ef86f983c47fd45457968b6bae82a35108096fcdc41` and Linux `a3789dc4bd592c8d4a930583f8f3f17a6ccdba150fba4e26605cc6d7a5e5c7a8`. The real extracted-package runner cell `C:\shared\src\incursa\.artifacts\tls-mtls-runner-evidence\tls-mtls-final-v5-direct-package-cell` completed `3,907 / 0 / 0` with zero application bytes, exact server and client DER/SPKI identities, exactly one transmitted client leaf, no transmitted client trust anchor, both chains verified, and no resumption or early data. The public Protocol Execution Result v2 schema passes and all `10/10` referenced artifact hashes verify.
- gRPC/H2 now has exact package and runner support for all twelve committed IDs: five unary shapes, three streaming shapes, three terminal outcomes, and fresh-channel unary echo. Clean `0.4.0` Windows package hashes are scenario `907a77f5d76fb876eb838c9695d766771c89d270fe8a4bf34aab7dd47354bf4a`, executor `1c92c2af2a6b0672803d0fb3372716420fe6a6c11c7fd5aa63bf233a887f5152`, and target `f3d4320304946e0f464c44978a5b7e2133524158d532a92b76ad5e74850b93bf`. The final terminal/channel evidence roots are under `C:\shared\src\incursa\.artifacts\grpc-completion-runner-evidence`; six Protocol Execution Result v2 documents pass schema validation and all `66/66` declared artifact hashes verify.
- HTTP/1.1 RFC 6455 now has exact runner-backed five-ID core coverage in both cleartext and TLS 1.3 lanes. Cleartext package hashes are scenario `b61b100bef5dac618865e6f4490a180ea72caaa5daed5caaf300b5ccb0f8854b`, executor `c6014ca4151b0f5263935a448bab3778606096ee4494cad055be92e519cb0437`, and target `b8803f56c39638b0832b003876fd6eef00ad8c574410e4413d4ab6761dd7bf54`. TLS package hashes are scenario `c2a83a34c1ed04e73b761f94a08c9d7651ce906b548a426732741651aaa7345e`, executor `c63d280b49a2421c8a5d13b73a4cd6f9f490b218380b4c2b35f6540a79f16683`, and target `99d36b3dd89dfc87f48bce64e92c8cd354c0cec9c82b83bec4b95d22b12494de`. All ten runner cells have zero failed and timed-out operations and valid Protocol Execution Result v2 evidence.
- DNS over QUIC now has exact package and runner support for `dns.doq.query.a`. The final extracted three-package runner cell completed `25,313` operations with zero malformed, retries, failures, or timeouts. Its Windows package hashes are scenario `6f640cdaf0059cec4b80f38147d388da62155ffc77ef74bf8bdd69bc4e282a0b`, executor `d5cead581dc1056cabee81781f84371d0456f6e86caca802b6b9fadb533aae38`, and target `d2fd918cc94ebfbda61540dca7463d72dc8292240b424c4f57eab1639fad75b0`. The public Protocol Execution Result v2 schema passes and all `12/12` referenced artifact hashes verify under `C:\shared\src\incursa\.artifacts\protocol-lab-internal-dns-doq-runner\dns-doq-runner-evidence\dns-doq-final-runner-v3-direct-package-cell`.

Final Linux package SHA-256 values are: TLS executor `b9c5ad8a9e5d2eafb4be2d234c1590c03144bfcfa6a3c5aa4bab4f02069a9ec2`; gRPC executor `2d87b50b6fc36e41f387db3d5a6d1ad6783dc0d62f25a7cb69de58dc4cc5ad4d`; gRPC target `e8503edae14776ff1a1084d65dfb23b937675e0b1eab7d5222597777bd6d82e7`; DoT executor `769c567ba580eaf946f457482558c8cc939f5c9b059b715c2f2691b8b1e647c2`; DoT target `2fe9dc1171850f216856736eea63ec002ae308e63f0ee032c14904bda849bfe0`; DoH2 executor `4cf8d425ad10d7667896b6c3c1b0735aaf7a72bbed1adc6b0ebf01726da64967`; DoH2 target `479f92a4170e49384fe612cf4501eef4c9e249481a2a37461e3518db1102035d`.

Runner admission is exact rather than family-substituting. Target manifests carry an exact `scenario.<id>` capability, and the runner requires that capability before using the package-executor role bridge. The TLS parser checks TLS version, cipher, key exchange, ALPN, certificate DER/SPKI identity, no resumption, no early data, zero application bytes, load echo, identities, outcomes, and artifacts. The gRPC parser additionally checks the committed service digest, HTTP/2/TLS proof, channel reuse, 128/131/136 byte scopes and hashes, status/media type/trailers, and all terminal counts. The DoT parser checks local-authoritative-only semantics, exact DNS question/answer/TTL, canonical 27/43-byte hashes, two-octet framing, message-ID correlation, TLS/ALPN identity, and separate malformed/retry/failure/timeout counts. The DoH2 parser independently requires TLS 1.3 with ALPN `h2`, HTTP/2 without fallback, POST `/dns-query`, exact request and response media/cache semantics, message ID zero, canonical response normalization, genuine connection-establishment timing, and the same local-authoritative-only DNS identity.

Explicit runner-unsupported inventories remain fail closed. TLS: `tls.early-data.accepted`, `tls.early-data.rejected`, and `tls.key-update.diagnostic`. gRPC/H2 has no remaining unsupported committed scenario ID in the `0.4.0` package and integrated runner. DNS: `dns.classic.tcp.query.a`, `dns.classic.udp-truncated-tcp-retry`, `dns.classic.udp.query.a`, `dns.doh3.get.a`, `dns.doh3.query.a`, `dns.doh3.query.aaaa`, `dns.doh3.query.cname-chain`, `dns.doh3.query.large-dnssec-shaped`, `dns.doh3.query.nodata`, and `dns.doh3.query.nxdomain`. WebSocket runner gaps remain all six RFC 8441 IDs and `http3.websocket.rfc9220.fragmented-binary-echo`; both now have exact component packages awaiting runner slices. The five RFC 9220 core IDs retain their existing runner support but must pass regression against the corrected exact-ID `0.2.0` component packages; no generic WebSocket substitution is allowed.

Verification commands included `go test -race -count=1 ./...` and `go vet ./...` for every affected Go executor and target; warning-as-error builds for the affected .NET runner and CLI projects; `Validate-ProtocolLabComponentManifests.ps1`; every affected package builder and build-attestation validator; exact extracted three-package runner smokes; public repository health validation; Protocol Execution Result v2 JSON Schema plus artifact-hash validation; and `git diff --check`. The recorded clean-source package archives have parity-eligible local attestations, but none has been published; all local smoke evidence remains diagnostic and non-publishable.

The installed `workbench validate --profile core` command is not a clean repository-local gate in this workspace: it exited `1` after traversing `C:\shared\src\incursa` and reporting pre-existing broken Markdown links in unrelated repositories, vendored `node_modules`, and other worktrees. This failure was not suppressed or represented as a ProtocolLab pass; the focused runner SpecTrace JSON parse, tests, builds, package validation, and public repository-health validator are the applicable green evidence for this slice.

## Architecture decision

Keep lane-scoped executor identities:

- `org.protocol-lab.components.executor.go-http1-executor` / `go-http1-executor`
- `org.protocol-lab.components.executor.go-http2-executor` / `go-http2-executor`

Share validation and normalization behavior where packaging permits, but do not use one executor ID for H1 and H2. Every lane must fail closed on protocol fallback, preserve its own unsupported states, and identify the selected load generator separately from the test executor.

The current internal runner collapses `testExecutorId` into `loadToolId` and special-cases arguments and parsers by tool ID. That makes the current Go H1 package validation-only and prevents a credible package-backed benchmark. The durable integration must separate these identities:

1. test executor: owns protocol/workload validity and result acceptance;
2. load-generator adapter: owns argument mapping, process execution, raw output, and tool identity;
3. parser/normalizer: owns tool-output normalization and parse failure;
4. runner: owns lifecycle, artifact capture, telemetry, provenance, and fail-closed orchestration.

Unknown executor or parser IDs must be `unsupported` or `unavailable`; they must never receive generic arguments and then be accepted with empty metrics.

## Current inventory

| Lane | Existing component truth | Readiness |
| --- | --- | --- |
| HTTP/1.1 target | Kestrel `0.1.2`; Caddy `0.1.0`; nginx `0.1.0` | Kestrel is the first proof target. Caddy/nginx require binary provenance hardening before ranking. |
| HTTP/1.1 executor | Published/base `go-http1-executor` `0.1.0`; this branch prepares diagnostic `0.3.0` | `0.3.0` proves exact H1, validates deterministic responses, embeds a pinned load generator, normalizes metrics, preserves raw artifacts, and has passed two local three-package smoke cells. It remains uncommitted and unpublished. |
| HTTP/1.1 load tool | Internal `oha` manifest/parser plus branch-local package-owned `oha 1.15.0` | The `0.3.0` diagnostic packages pin and SHA-256 verify official PGO binaries for `win-x64` and `linux-x64`; the internal runner bridge is still uncommitted. |
| HTTP/2 target | Kestrel `0.1.1`, h2c prior knowledge, plaintext/JSON only | Suitable for the first h2c proof; not TLS/ALPN coverage. |
| HTTP/2 executor | Branch-local `go-http2-executor@0.3.0` | Exact h2c prior-knowledge smoke, `1/8/8` diagnostic, and contract-backed `16/128/8` comparison shapes are implemented; streaming and TLS/ALPN remain unsupported. Comparison source proof is local and is not a ranking or publishability claim. |
| HTTP/2 load tool | Internal `h2load` plus branch-local package-owned `go-x-net-http2-h2c-load@0.1.0` | The custom engine is the accepted 1x1 smoke generator. `h2load` remains an unpinned internal tool and is not used by this vertical. |
| HTTP/3 | Multiple target packages; curl H3, h3spec, RFC 9220 executors | Specialized proof exists. No general HTTP application performance executor/scenario pack. |

The base internal runner already has `oha` and `h2load` argument builders and parsers, exact H2 request policy in its managed validity check, and raw stdout/stderr capture. The isolated E2 bridge adds package cwd/environment handoff, exact package executor identity, a normalized HTTP result parser, H1/H2 protocol-proof preservation, and separate executor/generator artifacts. h2c versus TLS/ALPN remains an execution-variant boundary; no TLS/ALPN result is claimed by the current H2 package.

## Delivery slices

### E0 — Inventory and isolation

Status: complete.

- Worktree: `C:\shared\src\incursa\.worktrees\protocol-lab-components-http-executors`
- Branch: `codex/http-executors-20260713`
- Base: local components commit `f61ed92`
- Non-goals: public contract edits, benchmarks, package publication, controller upload, deployment, lab changes, commit, or push.

Gate: component and internal authorities audited; dirty public/internal work preserved; exact package/runner gaps documented.

### E1 — HTTP/1.1 validity executor hardening

Status: complete on this branch and incorporated into the package-owned validation-plus-load executor used by E3/E4.

Files:

- `executors/go-http1-executor/source/main.go`
- `executors/go-http1-executor/source/main_test.go`
- `executors/go-http1-executor/test-executors/go-http1-executor.yaml`
- `executors/go-http1-executor/protocol-lab-package.json`
- legacy compatibility manifest and README

Required behavior:

- request cleartext HTTP/1.1 and record `resp.Proto`;
- reject HTTP/1.0, HTTP/2, redirects, TLS targets, and silent fallback;
- validate exact method/path, status, media type, payload length, bytes, and SHA-256;
- count failures and timeouts;
- emit `validation.json`, `result.json`, `protocol-proof.json`, and `executor-identity.json`;
- do not emit a fake load-tool record; metrics are emitted only after E3 load generation and validity both succeed;
- preserve raw stdout/stderr through the invoking host.

Acceptance:

- Go unit tests include positive exact-H1 and negative status/type/body cases;
- local Kestrel H1 smoke records observed `HTTP/1.1` with `fallbackDetected: false`;
- component manifests validate;
- diagnostic package build and package conformance pass;
- the package remains diagnostic/non-publishable even after E2–E4 local smoke passes.

### E2 — Minimum internal HTTP executor bridge

Owner: `protocol-lab-internal`, in a new isolated worktree from the current approved local base.

Status: implemented and targeted-test verified in isolated worktree `C:\shared\src\incursa\.worktrees\protocol-lab-http12-runner-bridge-clean` on branch `codex/http12-runner-bridge-clean-20260713`; not committed or integrated.

Required changes:

- honor package working directory, entrypoint, environment, and artifact directory;
- export target, protocol, scenario, artifact directory, duration, warmup, connections, concurrency, streams, repetition, and timeout context to the package;
- accept a declared normalized HTTP-executor JSON parser instead of defaulting unknown executors to `raw-quic-json`;
- reject a validation-only executor for a benchmark stage;
- reject unknown parsers, missing required metrics, empty parsed results, executor substitution, and load-generator substitution;
- record the selected executor and its underlying generator identity/version in artifacts and evidence;
- persist requested/observed H1/H2 version proof and execution variant;
- keep requested connections, concurrency, streams, warmup, duration, and repetitions separate from effective values;
- preserve stdout/stderr and command/version identity on success and failure.

Acceptance tests:

- unknown executor ID is unsupported, not generically invoked;
- validation-only executor cannot satisfy a benchmark cell;
- parser failure blocks metric acceptance;
- H1 fallback and H2 fallback fail;
- h2c and TLS/ALPN evidence cannot match;
- package-backed execution uses the materialized entrypoint and working directory;
- requested and observed executor/load-generator identities survive into evidence.

Observed verification:

- 153 focused materialization, wrapper, load invocation, contract-fixture, HTTP/3, and raw-QUIC regression tests pass after the direct-cell and executor-process sidecar changes;
- an earlier broader selected run passed 123 tests;
- the full test project is not green on the current base: 48 unrelated/baseline tests fail across architecture dependency policy, public run-plan fixture drift, and adapter executable availability;
- `git diff --check` reports only the pre-existing line-ending warning for `fixture-incursa-raw-quic-adapter-v1.yaml`.

The local benchmark wrapper now mirrors the C# materializer's fail-closed executor eligibility, parser, package cwd, and `executor-process/{executorId}.json` sidecar behavior. It also permits a direct package cell only when implementation, scenario, protocol, executor, and load-profile IDs are all explicit, avoiding a synthetic suite. This is the minimum bridge for the first H1/H2 vertical. A later evidence-hardening slice should make `testExecutorId`, `loadGeneratorId`, and parser identity first-class independent run-stage dimensions.

### E3 — Pinned HTTP/1.1 load-generator adapter

Owner: components after E2 interface approval.

Status: implemented as diagnostic `go-http1-executor@0.3.0` packages and locally runner-integrated on this branch; not committed, published, or comparison eligible.

Preferred first tool: `oha`, immutably pinned per supported runtime and recorded as execution metadata. The package may embed the pinned binary in the lane executor as the shortest compatible path; a separate toolchain package is acceptable only if E2 implements explicit dependency composition. Host `PATH` lookup and `latest` container tags are not comparison-grade.

Required normalized output:

- completed, failed, and timed-out requests;
- requests/second and bytes/second;
- latency mean and p50/p75/p90/p95/p99 where supplied;
- requested/effective connections and concurrency;
- duration and warmup;
- exact tool version, SHA-256, command, stdout, stderr, and parser status.

Gate: one validation pass followed by one timed load against `kestrel-http1@0.1.2`; any validity or parser failure rejects the metrics. This is local smoke evidence, not a ranking.

Observed verification:

- `go test -count=1 .` passes for the executor source;
- all 21 public and 21 internal component manifests validate;
- diagnostic packages build for `win-x64` and `linux-x64`, and dirty-source build attestations validate;
- packaged `oha` SHA-256 values match the pinned official `1.15.0` PGO assets;
- the final Windows diagnostic archive SHA-256 is `e0cebf7caaf23e5e72f9278eb877020343e68201d9962d54734c3e79e1aa2bdd` and reports `go-http1-executor 0.3.0` through the runner version probe;
- an extracted Windows package completed a local exact-H1 validation-plus-load smoke against Kestrel, with `fallbackDetected: false`, zero failed/timed-out operations, normalized metrics, and all declared raw artifacts;
- the smoke is executor/implementation diagnostic evidence only, not a benchmark or a three-package proof.

### E4 — HTTP/1.1 three-package vertical

Status: complete as a local, diagnostic, non-publishable vertical on this branch. The scenario package snapshots only tracked, unmodified, already-committed public authority at commit `a4dcd74e5c8907907ccc58808da92d2b920b2fbc` and records per-file hashes in `authority-lock.json`.

Packages:

- scenario: `org.protocol-lab.components.scenario.http1-performance@0.1.0`;
- executor: `org.protocol-lab.components.executor.go-http1-executor@0.3.0`;
- implementation: `org.protocol-lab.components.implementation.kestrel-http1@0.1.2`.

The scenario package may contain only approved public H1 scenario/suite/profile bindings. First scenarios are `http1.core.plaintext` and `http1.core.json`; `http1.payload.bytes.1kb` waits until target endpoint and deterministic-byte semantics align.

Gate: all three packages materialize with SHA-256 and build attestations; exact H1 validation passes; raw load artifacts and normalized metrics exist; evidence remains `local smoke` and non-publishable.

Observed verification:

- scenario package `org.protocol-lab.components.scenario.http1-performance@0.1.0` SHA-256 `cb4577002163b45deca09086e6b18d88d2b9765e23080dbe609f24987134f533` contains only `http1.core.plaintext`, `http1.core.json`, and `http1-smoke` authority snapshots;
- implementation package `org.protocol-lab.components.implementation.kestrel-http1@0.1.2` SHA-256 `0b6421385ed8c2444270c70d2163abd0433ca28502ef4939d7c1c744779ee1ac`;
- the plaintext and JSON cells each materialized exactly those three package identities, reported `HTTP/1.1`, `fallbackDetected: false`, matched deterministic status/type/length/SHA-256, recorded zero failures/timeouts, matched requested/effective `1/1/1`, and produced parsed metrics plus all required raw artifacts;
- the executor identifies the underlying generator independently as `oha@1.15.0` with the pinned Windows asset SHA-256;
- no H1 target or executor process remained after either run;
- the artifacts are under the short-lived local roots `C:\Users\Samuel\AppData\Local\Temp\plh1pr-bd20` and `C:\Users\Samuel\AppData\Local\Temp\plh1jr-bd20`; they are smoke evidence, not committed benchmark evidence.

### E5 — HTTP/2 h2c executor

Status: E5a 1x1 smoke and E5b 1x8x8 diagnostic executor, scenario package, and local three-package runner cells are complete on this branch. Comparison, multi-connection topology, streaming, and TLS/ALPN remain explicitly unsupported.

Packages:

- new `org.protocol-lab.components.executor.go-http2-executor@0.3.0`;
- new `org.protocol-lab.components.scenario.http2-performance@0.2.0`;
- package-owned `golang.org/x/net/http2` h2c load engine with logical generator identity `go-x-net-http2-h2c-load@0.3.0`;
- existing `org.protocol-lab.components.implementation.kestrel-http2@0.1.1`.

First supported scenarios: `http2.core.plaintext` and `http2.core.json`. Do not claim `http2.streaming.response` until the implementation exposes and validates its deterministic 64 KiB stream.

Required proof:

- h2c prior knowledge, never H1 upgrade/fallback;
- observed HTTP/2 response version;
- requested and effective clients/connections and max concurrent streams;
- exact endpoint/status/type/length/hash;
- raw generator summary plus normalized metrics;
- explicit unsupported outcome when the generator cannot prove the requested connection/stream behavior.

E5a supports the stable `http2-smoke` `1/1/1` shape at 5 seconds measured, 1 second warmup, and a 5-second per-request timeout. E5b adds only the committed `http2-diagnostic` `1/8/8` shape at 10 seconds measured, 1 second warmup, and a 10-second per-request timeout. The runner propagates the profile-authored timeout separately, and the executor fail-closes on a missing or mismatched value, enforces it per operation, and records it in requested/effective raw load evidence. The package counts actual dials and active operations, samples active and pending HTTP/2 streams, records peer-advertised `MAX_CONCURRENT_STREAMS`, and echoes requested/effective load independently. It rejects `http2-comparison`, TLS/ALPN, streaming response, or any other topology instead of deriving or approximating them.

Observed verification:

- `go-http2-executor@0.1.0` source tests pass, including direct h2c prior-knowledge, forced-H1 rejection, semantic rejection, topology proof, and unsupported-profile cases;
- all 24 public and 24 internal component manifests validate after adding the two scenario packages;
- complete diagnostic packages build for `win-x64` and `linux-x64`, contain a native executor, include exact `golang.org/x/net@v0.57.0` license text, and have validated dirty-source build attestations;
- the final Windows diagnostic archive SHA-256 is `c0122b802b45f5d2746dc9c650b9f5c401a2c9744aa9b3ea8948316d60f2c86c` and reports `go-http2-executor 0.1.0` through the runner version probe;
- an extracted Windows package passes against `kestrel-http2@0.1.1` with exact HTTP/2, no fallback, one observed dial, one maximum active stream, and zero failed/timed-out operations;
- forced HTTP/1 and concurrency `2` negative runs fail closed and emit no normalized metric envelope;
- scenario package `org.protocol-lab.components.scenario.http2-performance@0.1.0` SHA-256 `c8fa726ed3a5d64edbefba5bb8766fd44db4d97acfbd85696080f735a8c67c68` and implementation package `org.protocol-lab.components.implementation.kestrel-http2@0.1.1` SHA-256 `e8610239db6051cf0fee5d265f8f902645cb795b47d511eaafd6f1d7e5997662` both have validated attestations;
- the package-backed plaintext and JSON cells each materialized exactly three packages, proved `HTTP/2.0` over `http2-h2c-prior-knowledge`, reported no fallback, one observed dial, maximum active requests/streams of one, zero failures/timeouts, matched requested/effective `1/1/1`, and produced parsed metrics plus all required raw artifacts;
- the executor and logical generator identities remain separate: `go-http2-executor@0.1.0` and `go-x-net-http2-h2c-load@0.1.0`;
- no H2 target or executor process remained after either run;
- the artifacts are under `C:\Users\Samuel\AppData\Local\Temp\plh2pr-bd20` and `C:\Users\Samuel\AppData\Local\Temp\plh2jr-bd20`; this remains local smoke evidence, not a benchmark, comparison, or ranking.

E5b diagnostic verification:

- `go-http2-executor@0.2.0` and `go-x-net-http2-h2c-load@0.2.0` pass race-enabled source tests and `go vet`;
- diagnostic archives are `org.protocol-lab.components.executor.go-http2-executor.0.2.0.win-x64.plabpkg` SHA-256 `47339267d6608937b1602cba6b670bfea5c492d457edc5899c57cf4d87fdf9ae`, `org.protocol-lab.components.executor.go-http2-executor.0.2.0.linux-x64.plabpkg` SHA-256 `a1fecf8099b5e4391d417dc2006c111f8296827a062446aa4114786ae8d58820`, and `org.protocol-lab.components.scenario.http2-performance.0.2.0.plabpkg` SHA-256 `8a1e5df02af183ecdcc416cc9a5fbd56fdd9d4c7f92b6fad59ea53786208f5a8`, each with a matching dirty-source build attestation;
- the scenario package authority-locks tracked, unmodified `http2-diagnostic` from public commit `a4dcd74e5c8907907ccc58808da92d2b920b2fbc`;
- package-backed plaintext and JSON diagnostic cells each preserved requested and effective `connections=1`, `concurrency=8`, and configured `streamsPerConnection=8`, observed one dial, eight active operations, eight sampled active HTTP/2 streams, and a peer stream limit of 100;
- both cells proved exact HTTP/2, no fallback, deterministic status/type/length/hash, zero failed/timed-out operations, matched executor/load-generator identity, and an exact requested-load echo;
- artifacts are under `C:\Users\Samuel\AppData\Local\Temp\plh2diag2-0713` and `C:\Users\Samuel\AppData\Local\Temp\plh2diagjson-0713`; they remain same-host local diagnostic evidence, not comparison or publishable evidence.
- all four H2 package/build-attestation pairs validate, the components open-source audit reports all 12 required surfaces present, 172 focused internal runner/model/raw-QUIC-adjacent tests pass, and all five executable Incursa HTTP/3 adapter conformance checks pass after building the adapter project.
- request-timeout integrity release `go-http2-executor@0.2.1` / `go-x-net-http2-h2c-load@0.2.1` passes race-enabled source tests and `go vet`; the Windows archive SHA-256 is `964e13d72bbf274d77929bc2983c42b73f706b1942faabf1385b7b54e57fed8e` and the Linux archive SHA-256 is `7f8cc6833cf7a04f5908b7cc494149a5bcf5829a1b2139111a4c4bce32b6bb5a`, each with a dirty-source build attestation;
- 79 focused internal bridge/load-profile tests pass after adding `PLAB_REQUEST_TIMEOUT_SECONDS` propagation;
- the fresh package-backed plaintext diagnostic cell under `C:\Users\Samuel\AppData\Local\Temp\plh2timeout-0713` materialized exactly executor `0.2.1`, scenario `0.2.0`, and Kestrel `0.1.1`; it recorded request timeout 10 seconds in both requested/effective raw executor load, exact h2c/no fallback, one dial, eight active operations/streams, peer limit 100, 29,709 successful requests, zero failures/timeouts, accepted parser and executor identity, and matching requested load. It remains local diagnostic evidence only.

The public contract decision is now `16` simultaneously established connections, `128` globally in-flight operations, an `8`-stream per-connection cap, and `balanced-round-robin` assignment. The comparison suite contains only `http2.core.plaintext` and `http2.core.json`. Version `0.3.0` implements and source-tests that shape, emits requested/effective topology and per-connection proof in `http2-topology.json`, and keeps executor and generator identities distinct. A package-backed runner cell, second comparable origin target, controlled topology, telemetry/saturation proof, and matched repetition policy are still required before any comparison or ranking claim.

Remaining model gap: `Incursa.ProtocolLab.Model@1.0.8` still omits concurrency from `MatrixOptions` and `RunCell`. The runner now preserves profile-authored concurrency at execution-plan time and verifies the package echo, which is sufficient for committed profiles. Independent CLI concurrency overrides and comparison-group identity still require a durable model-authority decision. For E5b, `streamsPerConnection` is treated as configured per-connection capacity while sampled active streams are recorded separately.

### E6 — HTTP/2 TLS/ALPN executor mode

Add a distinct execution variant to the H2 executor only after h2c is stable. Require HTTPS, negotiated ALPN `h2`, certificate policy and chain metadata, no certificate bypass for publishable evidence, and explicit rejection of H1 ALPN fallback. Target packages need new immutable TLS-capable variants; current `kestrel-http2@0.1.1` remains h2c-only.

### E7 — HTTP/3 compatibility pass

Status: read-only compatibility audit complete after the H1/H2 package smokes; no H3 component or runner refactor was made.

- bundled `managed-httpclient-h3-load` remains on its existing managed path, retains `RequestVersionExact`, and does not consume the new package executor sidecar;
- the E2 parser additions are opt-in by parser ID, so existing `managed-httpclient-h3-json`, `h2load`, `oha-json`, and `raw-quic-json` parsing is unchanged;
- focused tests covering HTTP/3 and raw QUIC remain green in the 153-test selected run;
- package cwd/environment handoff can support a future general H3 executor and records `http3-quic-tls` as its execution variant, but the same independent-concurrency model limitation must be resolved first;
- existing `curl-http3-client`, `h3spec-http3-qpack`, and `aioquic-rfc9220-websocket` packages are specialized validation/proof executors, do not declare a normalized performance parser/load generator, and therefore cannot satisfy the new benchmark load-tool gate. This is an explicit compatibility gap, not a reason to default them to `raw-quic-json` or report empty metrics;
- no general package-backed HTTP/3 application performance executor/scenario vertical exists yet; the existing managed lane is not a substitute for package-backed provenance;
- raw QUIC remains a separate transport lane and its parser/fixture tests pass unchanged.

Next H3 action is a separate design slice: either package the existing managed H3 engine behind the normalized executor envelope or create a lane-scoped package that preserves exact QUIC/TLS/HTTP3 proof. Keep h3spec and RFC 9220 executors diagnostic/specialized.

### E8 — Later protocol executors

Activate only after the E2 interface and at least one HTTP vertical are stable:

1. TLS 1.3 full handshake, resumed handshake, then record throughput. Keep 0-RTT separate.
2. gRPC over HTTP/2 unary small/fixed payload, then server streaming and sustained duplex.
3. secure DNS common query semantics with separate DoT, DoH2, DoH3, and DoQ bindings against deterministic local authority data.
4. WebSocket RFC 6455, RFC 8441, and retained RFC 9220 bindings with distinct executor modes.

Each family requires its own exact protocol/session proof, deterministic response semantics, raw tool artifacts, explicit unavailable/unsupported states, and a local package-backed smoke before performance evidence is accepted.

### E8 preflight and proposed verticals

The public files exist, but they are still `draft` scenarios with `experimental` profiles and have executor-blocking fixture ambiguities. “Contracts exist” is not sufficient authority to invent those details in components.

#### E8a — TLS 1.3 full handshake first

Status: executor and independent target source are implemented, package manifests validate, and a direct local smoke passes. Package production and internal runner admission remain diagnostic/integration work; no scenario package, benchmark, comparison, or publication claim is made.

Proposed package identities:

- scenario: `org.protocol-lab.components.scenario.tls13-handshake-performance@0.1.0`;
- executor: `org.protocol-lab.components.executor.go-tls13-executor@0.1.0`, executor ID `go-tls13-executor`;
- logical generator: `go-crypto-tls13-handshake-load@0.1.0`;
- first target: `org.protocol-lab.components.implementation.dotnet-sslstream-tls13@0.1.0`, implementation ID `dotnet-sslstream-tls13`, role `library-backed-target`.

The first vertical selects only `tls.handshake.full` with committed profile `tls-smoke`: TLS 1.3 only, full authenticated handshake, no session reuse, no early data, zero application-data bytes, ALPN `protocol-lab-tls`, one connection/concurrency/handshake per connection, five-second duration, one-second warmup, and one repetition. `tls.handshake.resumed` and `tls.record.throughput` remain unsupported. A .NET `SslStream` target and Go `crypto/tls` executor keep the target and generator implementations independent.

Minimum executor proof: requested/observed TLS version, no downgrade, negotiated cipher suite, key exchange group, signature algorithm, ALPN, certificate chain/profile hash, full-versus-resumed state, completed/failed/timed-out handshakes, latency percentiles, exact executor/generator identity, stdout/stderr, `tls-negotiation.json`, `protocol-proof.json`, and normalized result. It must reject TLS 1.2, missing/wrong ALPN, untrusted or mismatched fixture identity, resumption, 0-RTT, and substituted executors.

Resolved contract decisions:

- `tls-smoke` is the only first-wave profile and binds only `tls.handshake.full`; it requires TLS 1.3, no resumption, no early data, zero application bytes, ALPN `protocol-lab-tls`, one simultaneous connection, one handshake per connection, five measured seconds, one warmup second, one repetition, and a five-second timeout;
- `plab-single-leaf-p256-v1` is a certificate fixture identity, not another scenario. The public fixture fixes the DNS identity `tls.protocol-lab.test`, ECDSA P-256/SHA-256 leaf and root, chain shape, serial, validity interval, and DER/SPKI hashes while keeping the private key out of public authority;
- the primary handshake latency begins only after TCP is established and ends at authenticated TLS completion. DNS, TCP setup, process startup, certificate generation, and artifact writing are excluded. Connection-plus-handshake duration from TCP connect start is optional diagnostic metadata;
- package-owned normalized non-HTTP validity is recommended for the recognized TLS executor, with the internal runner still required to fail closed on parser, identity, protocol, certificate, ALPN, session-state, artifact, or metric mismatch.

Implemented local evidence:

- `go-tls13-executor@0.1.0` uses Go `crypto/tls`, trusts only the fixed public root, requires exact TLS 1.3 and ALPN, checks the leaf DER/SPKI hashes, rejects resumption, performs no application I/O, normalizes handshake metrics, and preserves negotiation, proof, topology, latency-sample, identity, result, stdout, and stderr artifacts;
- `dotnet-sslstream-tls13@0.1.0` uses .NET `SslStream` as an independent library-backed server target and packages the test-only private key as implementation material, never as public authority or evidence;
- source tests, race tests, `go vet`, .NET build, and a Windows direct smoke pass. The smoke completed 2,425 full handshakes with zero failures and zero timeouts, but it is same-host diagnostic evidence rather than a benchmark.
- diagnostic archives now build and validate with dirty-source attestations: Windows executor SHA-256 `d4ad537ea3f2d6d8cdcecf0ac96b14a2d03e5f1c5e7395e52e133df803b4c48b`, Linux executor SHA-256 `47c076b8b728c6ce4cd805163008a1b497ba0ec30efaa765bd59e373cb093534`, and portable .NET target SHA-256 `f60886c4d8bc88a0aeed9064bfb95ec17e513c087ec0a0cc4180fb9567f9ae9d`;
- an extracted Windows executor/target package smoke completed 2,623 full TLS 1.3 handshakes at the requested `1/1/1` topology with zero failures and zero timeouts, exact `protocol-lab-tls` ALPN, `didResume: false`, the fixed leaf hashes, and all 13 executor artifacts. A direct evidence scan found no private-key text; the target archive alone contains the explicitly test-only private key.

Completion update: `tls.handshake.resumed` is complete as a clean-source,
package-backed local diagnostic vertical. Component source commit `8bdaa67` and
runner admission commit `43f0e9c` preserve the existing full-handshake lane
while adding exact accepted single-use PSK resumption. The scenario package is
locked to public authority commit
`8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574`.

Clean package SHA-256 values:

- scenario `org.protocol-lab.components.scenario.tls13-handshake-performance@0.1.0`: `bdb24e1ced0f6283fffa5010d58e5747ad80a1c39405d4e181552b7cc15d715e`;
- executor `org.protocol-lab.components.executor.go-tls13-executor@0.2.0`: Windows `523fc97bf465e770aeb47b7d38ef13017fc2f4493245f0c9f1e1fd1302340de9`, Linux `4504d5c573bf8bd08f5b95ab744c6b5031c9424d4b561b77d8505e930c01ca69`;
- target `org.protocol-lab.components.implementation.go-tls13@0.1.0`: Windows `69e1dbf8964a246b059cdf103728eb61bc15c8dc9141cd1a9918249fa82c44ac`, Linux `2e3d9feca1a0fe03f828c7142140ae1f2ef97cb1399572343c1b90a094aaee21`.

The clean extracted component smoke completed `6448` resumed handshakes with
zero failures/timeouts. The final real runner cell
`tls13-resumed-clean-v2-direct-package-cell` completed `8097` operations with
zero failures/timeouts. It proved exact TLS 1.3, X25519,
`TLS_AES_128_GCM_SHA256`, ALPN `protocol-lab-tls`, authenticated DER/SPKI
hashes, `didResume: true`, one unmeasured source full handshake and one
single-use ticket per measured operation, no warmup-state reuse, no early data,
zero application bytes, and exact executor/generator identities. Protocol
Execution Result v2 passed the public Draft 2020-12 schema and all `12/12`
referenced artifact hashes recomputed successfully. Evidence is under
`C:\shared\src\incursa\.worktrees\protocol-lab-tls-resumed-runner\artifacts\tls-resumed-runner-evidence\tls13-resumed-clean-v2-direct-package-cell`.

Exact runner invocation:

```powershell
pwsh -NoLogo -NoProfile -File scripts\benchmarking\Invoke-ProtocolLabBenchmarkSet.ps1 `
  -ImplementationIds go-tls13 `
  -ScenarioIds tls.handshake.resumed `
  -Protocol tls `
  -OverrideLoadToolId go-tls13-executor `
  -OverrideLoadToolMode process `
  -LoadProfileId tls-smoke `
  -ComponentPackageDirectory C:\shared\src\incursa\.worktrees\protocol-lab-components-tls-resumed\artifacts\tls-resumed-clean-packages `
  -ComponentPackage 'org.protocol-lab.components.implementation.go-tls13.0.1.0.win-x64.plabpkg,org.protocol-lab.components.executor.go-tls13-executor.0.2.0.win-x64.plabpkg,org.protocol-lab.components.scenario.tls13-handshake-performance.0.1.0.plabpkg' `
  -ComponentPackageMaterializationRoot C:\shared\src\incursa\.plab\tls-resumed-runner-clean-v2 `
  -RunIdPrefix tls13-resumed-clean-v2 `
  -Output C:\shared\src\incursa\.worktrees\protocol-lab-tls-resumed-runner\artifacts\tls-resumed-runner-evidence `
  -Configuration Debug -NoRestore -NoBuild -FailOnError
```

The same executor recognizes
`tls.handshake.full.tls12`, `tls.handshake.full.chacha20`,
`tls.handshake.mutual-auth`, `tls.early-data.accepted`,
`tls.early-data.rejected`, `tls.key-update.diagnostic`,
`tls.record.coverage`, and `tls.record.throughput` as explicit
`protocol-lab.unsupported.v1` outcomes with exit code `3`; none aliases the
full or resumed scenario. These local smokes remain diagnostic and
non-publishable.

Completion update: `tls.handshake.full.chacha20` now has a narrow package-local
vertical locked to public authority commit
`8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574`. The scenario package retains
`org.protocol-lab.components.scenario.tls13-handshake-performance@0.2.0`. The
test-side package is truthfully identified as
`org.protocol-lab.components.executor.go-utls-tls13-chacha20-executor@0.1.0`
with executor `go-utls-tls13-chacha20-executor@0.1.0` and logical generator
`go-utls-tls13-chacha20-load@0.1.0`. It pins
`github.com/refraction-networking/utls@v1.8.2` and applies a custom ClientHello
that offers only TLS 1.3, `TLS_CHACHA20_POLY1305_SHA256`, X25519,
`ecdsa_secp256r1_sha256`, and ALPN `protocol-lab-tls`, with no ticket, PSK, or
early-data extension. The independent target remains
`org.protocol-lab.components.implementation.go-tls13-chacha20@0.1.0`, built on
Go `crypto/tls`; the target and load generator are not the same implementation.

Extracted Windows and WSL2 Linux package smokes each completed one validation
handshake with zero failures and zero timeouts. Both proved the exact suite,
X25519, ALPN, canonical DER/SPKI certificate hashes, `didResume: false`, no
session-state or early-data offer, and zero application bytes. All nine other
committed TLS identities return explicit `unsupported`; an unknown identity
fails closed. Clean component source commit
`61af6713b99149dcc29fe1d22d35977cfec78e89` produced parity-eligible packages:
portable scenario
`92a317509a7d5e25a9145c4b53b697254a6a3c9e14e4e4b9464379b91a13b083`;
Windows executor
`7e4307713f4ba2614e124f552cc722e6c150b18bd3ecb43fad9885c7a80f0bb9`
and target
`76e5855adbdc3a42368152f053f3403772664cfe1960b13972612bc0bdbe82ae`;
Linux executor
`f141d1e904e68fdfa0102557fae0c8e4540e7e8d9e722833510be73648e3953b`
and target
`736d25dc5df8dc2c6a7fe7ba5d2771df81c3768d2ea261e3df6e2b876a924a77`.
The exact extracted evidence roots are
`artifacts/tls13-chacha20-clean-win-smoke` and
`artifacts/tls13-chacha20-clean-linux-smoke` within the isolated component
worktree. These local cells remain diagnostic and non-publishable; no generic
uTLS, TLS, comparison, or ranking support is implied.

The real extracted-package runner cell is
`C:\shared\src\incursa\.artifacts\tls13-chacha20-runner-evidence\tls13-chacha20-final-v1-direct-package-cell`
at runner integration commit `5bad8ff`. It completed `3,188 / 0 / 0` with zero
application bytes, emitted a public-schema-valid Protocol Execution Result v2,
and verified all `10/10` referenced artifact hashes. Admission additionally
fails closed on substituted uTLS engine/version, single-suite ClientHello
policy, package identity, protocol, cipher, certificate, session state,
metrics, or artifacts.

#### E8b — gRPC over HTTP/2 unary after TLS/ALPN

Proposed package identities:

- scenario: `org.protocol-lab.components.scenario.grpc-h2-performance@0.1.0`;
- executor: `org.protocol-lab.components.executor.go-grpc-h2-executor@0.1.0`, executor ID `go-grpc-h2-executor`;
- logical generator: `go-x-net-http2-grpc-load@0.1.0`;
- first target: `org.protocol-lab.components.implementation.go-grpc-h2@0.1.0`, implementation ID `go-grpc-h2`.

The first vertical should select only `grpc.h2.unary.echo` with `grpc-h2-smoke`: exact TLS/ALPN `h2`, channel reuse, identity compression, empty user metadata, one concurrent RPC, deterministic request/response payload, HTTP status 200, `application/grpc`, and `grpc-status: 0`. Streaming scenarios remain unsupported until unary proof is stable.

Completion update: public gRPC service v2 JSON at authority commit `8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574` is the canonical language-neutral authority. The package-local `.proto` is implementation material and its parity test locks package, service, message, method, streaming, status, compression, and metadata semantics to canonical digest `b7b987814f8af5cd4f15c03989b9c309c1c0ec643972ae32668304d71502120f`. Exact TLS 1.3/ALPN `h2`, no fallback, channel reuse, byte-scope hashes, and Protocol Execution Result v2 are proven by the local three-package cell recorded above.

#### E8c — secure DNS starting with DoT

Proposed first package identities:

- scenario: `org.protocol-lab.components.scenario.dns-dot-performance@0.1.0`;
- executor: `org.protocol-lab.components.executor.go-dns-dot-executor@0.1.0`, executor ID `go-dns-dot-executor`;
- logical generator: `go-dns-dot-load@0.1.0`;
- first target: `org.protocol-lab.components.implementation.go-dns-dot@0.1.0`, implementation ID `go-dns-dot`.

DoT is the first implementation slice because it reuses TLS 1.3 without also introducing an HTTP or QUIC binding. This is delivery sequencing only: DoT, DoH2, DoH3, and DoQ remain four independently selected, negotiated, validated, packaged, and reported transport lanes. Future bindings require separate executor/package identities while allowing a shared package-internal deterministic DNS semantic engine.

Minimum proof for every transport: exact transport/ALPN/version, no fallback, query ID echo, question/rcode/answer/TTL, deterministic normalized wire hash and length, connection/session reuse, completed/malformed/retried/failed/timed-out operations, latency/throughput, and raw DNS plus transport artifacts. Public recursive resolvers and external upstream lookups remain ineligible.

Completion update: `dns.plab-test-a.canonical` is committed public authority at `8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574`. The DoT scenario package authority-locks its exact 27-byte query, 43-byte authoritative answer, transport framing, and message-ID normalization contract. The local three-package DoT cell is complete as recorded above. DoH2 has also closed through its independent package and runner lane.

The DoQ slice provides `org.protocol-lab.components.scenario.dns-doq-performance@0.1.0`, `org.protocol-lab.components.executor.go-dns-doq-executor@0.1.0`, and `org.protocol-lab.components.implementation.go-dns-doq@0.1.0`. It selects only `dns.doq.query.a` with `dns-doq-performance-smoke` and `secure-dns-smoke`; locks the exact scenario, suite, profile, canonical DNS fixture, and TLS profile bytes to the public authority commit; and keeps QUIC/DoQ distinct from DoT, DoH2, DoH3, UDP, and TCP. The target is a fixture-backed local authority with no recursive or external upstream. The executor and runner require QUIC v1, TLS 1.3, ALPN `doq`, no resumption or 0-RTT, one query per client-initiated bidirectional stream, both FINs, message ID zero with identity hash normalization, exact 27/43-byte DNS hashes and 29/45-byte framed lengths, the fixed certificate hashes, exact executor/generator identities, and separate completed, malformed, retried, failed, and timed-out counts. Other committed DNS scenario IDs return explicit `unsupported`; unknown identities fail closed. This is local diagnostic support and makes no benchmark, comparison, publication, or generic QUIC-executor claim.

Clean packages built from components commit `0ba643fe5fe3670cf17d140782d04b1107715cbc` have parity-eligible attestations. Windows SHA-256 values are executor `d5cead581dc1056cabee81781f84371d0456f6e86caca802b6b9fadb533aae38` and target `d2fd918cc94ebfbda61540dca7463d72dc8292240b424c4f57eab1639fad75b0`; Linux values are executor `5f8644f0560a81e75e36ceefc402673705662c9259a965547ebb940d8fa9afe8` and target `17b6c1ef580a9d0200b8bf3af00aa67b5b46df33c73943a8ca5bcf12f55715d2`; the portable scenario package is `6f640cdaf0059cec4b80f38147d388da62155ffc77ef74bf8bdd69bc4e282a0b`. A clean extracted Windows package-local smoke completed 25,099 operations with zero malformed, retries, failures, or timeouts and verified all twelve other DNS identities as unsupported. The final real runner cell completed 25,313 operations with the same zero outcome counts, emitted schema-valid Protocol Execution Result v2, and verified all 12 referenced artifact hashes at runner commit `5a84ffb4ace47f88df16fe87d99e8fcd9220c560`.

The classic-DNS components-only slice is also complete at source commit `d48369414acaf10b563df8c83736ab5d8fe102f4`. `org.protocol-lab.components.scenario.dns-classic-calibration@0.1.0` locks the three exact scenario files, `dns-classic-calibration-diagnostic-smoke`, `dns-classic-diagnostic`, and both v2 wire fixtures byte-for-byte to public authority commit `8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574`. The target is `org.protocol-lab.components.implementation.go-dns-classic-authority@0.1.0`; it binds local UDP and TCP only, serves no recursive, cached, or upstream data, and supports only the v2 A fixture plus the 45/630-byte large truncation fixture. UDP and TCP remain distinct executor, parser, protocol-proof, artifact, and future runner-evidence lanes: `org.protocol-lab.components.executor.go-dns-udp-executor@0.1.0` owns UDP A and the contract-mandated UDP-TC-to-TCP retry, while `org.protocol-lab.components.executor.go-dns-tcp-executor@0.1.0` owns only TCP A.

All seven clean packages have parity-eligible attestations. SHA-256 values are scenario `61530690db9b4fb72d8ad354b7d72d11b36094a21dcbb1b9037bf421e9ead1d5`; target Windows `b7c65ef69dc0b9bc4330e9279e497791933228102793f0ce01807cb4bcbc8592`, target Linux `d13a5d24b55dbd320a95cf63dc03769a86e2b36bfa9348781eaf884aa9ce1fe0`; UDP executor Windows `0cd738784aa213feecd1791ee454412f05e4a78e3c2722338d7934dfd2506ee0`, UDP executor Linux `e9a830f0f85b07bac5727d0c88180e7d4351260ec914f8bc605588ed04801d8f`; TCP executor Windows `08418967f2575e4012241bed84abcabae966a958cafd21e16dd9e466e8094a11`, TCP executor Linux `6041883945d6deee1c182d8c516a0774165d2269723d521036f49cdd66628dde`. Source manifests and every extracted archive pass package-v2 identity/entry conformance, and every attestation passes the parity-eligible validator.

The clean extracted Windows proof is under `C:\shared\src\incursa\.worktrees\protocol-lab-components-dns-classic\artifacts\dns-classic-three-package-smoke-clean-d483694`. UDP A completed `56,735` operations, TCP A completed `75,433`, and UDP truncation-to-TCP completed `31,120`; every cell recorded zero malformed, failed, and timed-out operations. The retry cell recorded exactly `31,120` contract retries and its per-operation proof requires an advertised UDP size of 512, a 45-byte TC response, one identical-question persistent-TCP retry with a new unique ID, and the 630-byte canonical response with prefix `0276`. Runtime IDs are unique among outstanding operations and normalized to zero before all canonical query/response hash checks. Each cell preserves validation, protocol proof, DNS wire summary, normalized result, warmup/load summaries, executor/load-generator identities, and redirected stdout/stderr. Both executors return explicit `unsupported` for every other committed DNS identity and reject unknown identities with exit code `2`. This remains local diagnostic component evidence: no runner admission, Protocol Execution Result v2, comparison, publication, or ranking claim is made.

#### E8d — HTTP/2 RFC 8441 WebSocket

Component delivery selects a raw, independently implemented pair rather than
an API that hides the binding. The test side is
`org.protocol-lab.components.executor.go-http2-websocket-executor@0.1.0`,
executor `go-http2-websocket-executor@0.1.0`, and generator
`go-x-net-http2-websocket-load@0.1.0`, pinned to
`golang.org/x/net/http2@v0.57.0` raw Framer and HPACK APIs. The target is
`org.protocol-lab.components.implementation.kestrel-http2-websocket@0.1.0`,
which uses .NET 10 Kestrel `IHttpExtendedConnectFeature.AcceptAsync` and a
package-local raw WebSocket frame parser. The scenario package
`org.protocol-lab.components.scenario.http2-websocket-performance@0.1.0`
authority-locks all six exact identities plus their suites and load profiles to
public commit `8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574`.

The direct and extracted-package technology proof requires TLS 1.3, ALPN `h2`,
server `SETTINGS_ENABLE_CONNECT_PROTOCOL=1`, exact CONNECT pseudo-headers,
status 200, absence of HTTP/1.1 and Sec-WebSocket-Key/Accept headers, client
masking, deterministic message bytes and hashes, strict 100-message ordering,
ping/pong payload equality, and clean code-1000 close. HTTP/1.1 and HTTP/3
WebSocket identities return explicit unsupported evidence; unknown identities
fail closed. Evidence remains local, diagnostic, and non-publishable.

Clean component source commit
`6511ea4bcdd91d192e6f7cca65e53ab4f5a0816b` produced five
parity-eligible package attestations. The scenario archive SHA-256 is
`f25473baa73690634cf03a4c703a04e2c7b7f23362b4d1a15d4a5eacad76be1a`;
executor archives are
`be123f0fb6155e01716c5bcfc56d4f2bc6ad9bfcc68b488337cec59bdd92a969`
for Windows and
`c1912c488f95e59f535851f3f102e6a333e0250ecb8423e0304152eb66dcb18b`
for Linux; target archives are
`cc1ef78b9fecce05fd4d1034563acdcf8599d07133fbf75eb7c3f10fa548f92b`
for Windows and
`3b9dd0d228d5549a5c92b24953c24d471c3973cb5bf4cd5512ae3a173cd81a47`
for Linux. All lock public authority commit
`8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574`.

Extracted Windows evidence is under
`artifacts/http2-websocket-clean-6511ea4-win-smoke`; extracted Linux evidence
is under `artifacts/http2-websocket-clean-6511ea4-linux-smoke`. Each of the six
exact identities completed one independently selected validation operation
with zero failed and zero timed-out operations on both operating systems. The
ordered cell proved 100 messages, the target logs proved six masked-client and
six clean-close observations per operating system, all 18 adjacent HTTP/1.1
and HTTP/3 identities returned explicit `unsupported` in the Windows package
smoke, and the unknown-ID probe failed closed. These are local diagnostic
component results, not runner admission, publishable evidence, or rankings.

#### E8e — HTTP/1.1 WebSocket over TLS 1.3

Status: the original five-ID `0.1.0` package and runner lane remains complete. Component version `0.2.0` adds two individually routed diagnostics with source, extracted-package, exact runner-admission, and Protocol Execution Result v2 proof.

- scenario: `org.protocol-lab.components.scenario.http1-websocket-tls-performance@0.2.0` authority-locks seven exact `http1.websocket.rfc6455.tls.*` identities, the unchanged five-ID suite `http1-websocket-tls-performance-smoke`, and `websocket-smoke` to public commit `8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574`; the two diagnostics are not silently inserted into the canonical suite;
- executor: `org.protocol-lab.components.executor.go-http1-websocket-tls-executor@0.2.0`, executor `go-http1-websocket-tls-executor@0.2.0`, and generator `go-http1-websocket-tls-load@0.2.0`;
- target: independent `org.protocol-lab.components.implementation.go-http1-websocket-tls@0.2.0`, implementation `go-http1-websocket-tls@0.2.0`;
- base proof remains exact TLS 1.3, SNI `websocket.plab.test`, ALPN `http/1.1`, authenticated package-local DER/SPKI hashes, `didResume: false`, no early data, exact HTTP/1.1 status-101 Upgrade headers, fresh random 16-byte keys with zero reuse or accept mismatch, deterministic messages, and clean code-1000 close;
- diagnostic proof adds exact `plab.echo.v1` offer/acceptance and exact `permessage-deflate; client_no_context_takeover; server_no_context_takeover` offer/acceptance. Compression evidence requires correct masking, binary opcode, RSV1 in both data directions, semantic decompression, and the canonical 1 KiB payload hash; it deliberately does not require identical compressed wire bytes;
- unsupported: cleartext substitution, TLS 1.2, adjacent subprotocols or compression parameters, RFC 8441, RFC 9220, fragmentation, WebTransport, `websocket.echo`, and unknown identities fail closed rather than reusing a supported diagnostic;
- clean immutable packages at source commit `ad2fb8d1619c2c8858ade730dc7dd974ad2b7666`: scenario SHA-256 `1a563bd2b0b01d0372a4f200fef140c42b1827a6e66a704ed24eaa96596b4127`; executor Windows `8184071574c5a4204452d2ea6ac3db58a44fb979cce7911582ebbc604f976a7e`, Linux `6af0758ced17b35489042a6b7be823c0c038db340c9c205cdc1ac2b0e8742e50`; target Windows `aa0d8297d78ed56d0bd7aa0af36d3352db0e3cc0c1815a8de5bea5a16e6fee7c`, Linux `35271a7580bcf6f39ecc5ee9a55de4a3ed31d3a732a7284011c57ccb327a8edf`. All five build attestations report clean source and pass the parity-eligible validator;
- extracted Windows proof from those clean archives completed subprotocol text echo `54,871` and permessage-deflate binary echo `7,758`, plus the unchanged regression: upgrade `1,818`, control frames `75,836`, text echo `76,653`, binary echo `60,892`, and close `2,427`. Every cell recorded zero failed and timed-out operations; each smoke also returned explicit unsupported evidence for all 18 adjacent exact identities and rejected the unknown probe with exit code `2`;
- runner proof at commit `d1cdde34e7f4644ab2c5cb3f18fd20167c94d65e`: subprotocol `75,324` completed and deflate `9,063`, both with zero failures/timeouts; both public Protocol Execution Result v2 documents pass schema validation and all `16/16` artifact hashes verify. The separate `0.1.0` regression completed upgrade `2,712`, control frames `84,533`, text echo `79,731`, binary echo `69,523`, and close `2,686`, all with zero failures/timeouts and schema/hash-valid evidence;
- artifact roots: `C:\shared\src\incursa\.artifacts\http1-websocket-tls-diagnostics-clean-packages`, `C:\shared\src\incursa\.artifacts\http1-websocket-tls-subprotocol-clean-smoke`, `C:\shared\src\incursa\.artifacts\http1-websocket-tls-permessage-deflate-clean-smoke`, `C:\shared\src\incursa\.artifacts\http1-websocket-tls-five-id-clean-regression`, and `C:\shared\src\incursa\.artifacts\http1-websocket-tls-diagnostics-runner-evidence`. This remains local diagnostic evidence only; comparison, publication, and ranking are not implied.

#### E8f — HTTP/3 RFC 9220 fragmented binary diagnostic

The immutable authority pack is
`org.protocol-lab.components.scenario.http3-websocket-performance@0.2.2`.
It locks all six exact public scenario files plus `websocket-smoke` and
`diagnostic` byte-for-byte to public commit
`8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574`. The corrected runtime packages are
`org.protocol-lab.components.executor.aioquic-rfc9220-websocket@0.3.0` and
origin-server target
`org.protocol-lab.components.implementation.aioquic-http3@0.3.0`. The older
`0.2.1` executor/target package evidence is superseded and is not used for
admission or completion claims.

Clean source commit `ba59685252338576315e722ab561106ef0d91701`
produced the portable package SHA-256 values scenario
`dc052b415227dc30b42a14e5590de096e82f7a4dc7e15a9cce1bb5dda90ad28d`,
executor `9c09c853d65aff7fd3868f7f754ce096afdc8b6da0376ebafdb86aa64a8e7e60`,
and target `96d9c166ae58c7e290aff76e5efd3b8b66916cff2916a8b5bdb4d2fc8dd34965`.
Clean runtime-specific archive hashes are executor Windows
`952f9c9bb9b6af72a4d7b38d5b2dceeb3b11a804a16edf851b0f56497ef79a6c`,
target Windows
`09d1495ceacd1e1154f1579559f179ea1bd2de8e08a73feeb3dd63eb5c74ff23`,
executor Linux
`6bba10ffcab7e922b973b8ecf5422416f7733f0743e864f1b76eaa0daeeebd12`,
and target Linux
`757198329e642cc415c568f12420ba31d8c3f2b166608d8cdb4b0b09338d6b42`.
All seven archive/attestation pairs report clean source, pass the
parity-eligible attestation validator, and preserve source-to-extracted
authority parity.

The extracted three-package proof built immutable executor image
`sha256:96fc97c790a0ca4115e74f37efe6d866514fb146a5e490d379591813227cb23a`
and target image
`sha256:c687a4cea8aa7126eb3e8e91b6f845d0e9a7b6fcda2a0254a80f29b9d6a1bb1c`.
Every cell authenticated the fixed `websocket.plab.test` certificate with leaf
DER SHA-256
`fe996190f39355e3cfc201cbb7e2cba962a701b94ed08ff49e68e830216d0109`
and SPKI SHA-256
`c2440fbe955033f341ca625c1804e21b50066d952ab24a4b53007dc1cfbf410c`.
It also proved exact QUICv1, TLS 1.3, ALPN `h3`, HTTP/3 Extended CONNECT,
`SETTINGS_ENABLE_CONNECT_PROTOCOL=1`, pseudo-headers, no fallback or forbidden
WebSocket headers, client masking, deterministic payload/hash, clean code-1000
close, executor/generator/parser identities, origin-server role parity,
requested/effective load, archive/image materialization, and required raw
stdout/stderr and protocol artifacts.

The five core identities used one QUIC connection, concurrency and observed
active streams `1`, one-second warmup, and five-second measured duration.
Completed counts were extended CONNECT `661`, control frames `2,127`, text
echo `2,273`, binary echo `714`, and close `686`, all with zero failures and
timeouts. The fragmented diagnostic used one QUIC connection, concurrency and
observed active streams `8`, one-second warmup, ten-second measured duration,
one-second cooldown, and ten-second timeout. It completed `3,075` operations
with zero failures/timeouts and proved every message as exact masked fragment
payloads `[1024, 2048, 2928]`, binary/continuation/continuation opcodes, FIN
`false/false/true`, no interleaved control frame, ordered reassembly of 6000
`0xA5` bytes with SHA-256
`8f8d8f75d55c80475ffb0c12b1ede7083d6df689e8ef04f05176c5050873bfb7`,
and exact binary echo. Normalized results include `messagesPerSecond` and
`bytesPerSecond`; frame artifacts retain bounded samples plus total counts.

Because `aioquic-http3@0.3.0` is a shared target, focused regression proof also
passed its prior exact origin behavior: `/status` returned 22 deterministic
bytes, `/bytes/1024` returned the deterministic 1 KiB sequence,
`/headers/response?count=50&size=32` returned 50 exact workload headers, and an
unknown path remained an exact 404. The package does not broaden any scenario
claim, and raw QUIC, HTTP/1, HTTP/2, unproven larger payloads, and other known
cases remain explicit unsupported/unproven inventory.

Clean packages are under
`C:\shared\src\incursa\.artifacts\rfc9220-corrected-packages-20260713-v2`.
Extracted proof is under
`C:\shared\src\incursa\.worktrees\protocol-lab-components-rfc9220-corrected\artifacts\rfc9220-corrected-extracted-smoke-v2`.
This is local diagnostic component evidence only. The result shape and parser
metadata are runner-admissible inputs, but a real internal runner parser and
admission smoke remain separate work; publication, comparison, and ranking are
not claimed.

The remaining explicit `unsupported` identities are `websocket.echo`,
`http1.websocket.rfc6455.cleartext.upgrade`,
`http1.websocket.rfc6455.cleartext.control-frames`,
`http1.websocket.rfc6455.cleartext.text-echo`,
`http1.websocket.rfc6455.cleartext.binary-echo`,
`http1.websocket.rfc6455.cleartext.close`,
`http1.websocket.rfc6455.tls.upgrade`,
`http1.websocket.rfc6455.tls.control-frames`,
`http1.websocket.rfc6455.tls.text-echo`,
`http1.websocket.rfc6455.tls.binary-echo`,
`http1.websocket.rfc6455.tls.close`,
`http1.websocket.rfc6455.tls.subprotocol-text-echo`,
`http1.websocket.rfc6455.tls.permessage-deflate-binary-echo`,
`http2.websocket.rfc8441.extended-connect`,
`http2.websocket.rfc8441.control-frames`,
`http2.websocket.rfc8441.text-echo`,
`http2.websocket.rfc8441.binary-echo`,
`http2.websocket.rfc8441.close`, and
`http2.websocket.rfc8441.multi-message-text-echo`. No identity is substituted.

## Verification floor

For every components slice:

```powershell
pwsh ./scripts/package/Validate-ProtocolLabComponentManifests.ps1
python C:\Users\Samuel\.codex\skills\open-source-repo-maintenance\scripts\audit_repo.py C:\shared\src\incursa\.worktrees\protocol-lab-components-http-executors --profile incursa --format markdown
git diff --check
```

Also run package-local tests, the affected package builder, package-v2 conformance against the source and archive, and an extracted package smoke. Dirty-source builds are diagnostic-only and must use the explicit dirty-source option; they are not parity or publication evidence.

## Remaining owner decisions

1. Select the runtime-model authority for independent CLI concurrency and comparison-group identity before H2 comparison work. Preserve configured stream capacity separately from observed active-stream evidence.
2. Decide whether a future controlled TLS campaign must run both Windows and Linux hosts. Both executor and Go target archives are built; each approved host topology still requires controlled proof.
3. Continue the active all-identities goal through gRPC unary breadth, HTTP/1.1 WebSocket breadth, sustained/streaming lanes, diagnostics, classic-DNS calibration, and retained RFC 9220 proof. Each identity requires its exact committed semantics and may not alias a neighboring completed scenario.
4. Approve any controlled-lab campaign, package publication, controller upload, deployment, service restart, or lab-machine operation separately. Current evidence is local, single-host, diagnostic, and non-publishable.

Local source commits are authorized and recorded above. No push, package
publication, controller upload, service restart, deployment, public-site
update, benchmark campaign, or lab-machine change is authorized by this plan.
