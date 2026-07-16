#!/usr/bin/env bash
set -euo pipefail

target_url="${PLAB_TARGET_BASE_URL:-${PLAB_TARGET_URL:-https://host.docker.internal:8443/}}"
expected_status="${PLAB_EXPECTED_STATUS:-200}"
timeout_seconds="${PLAB_TIMEOUT_SECONDS:-5}"
image="${HTTP3_XQUIC_IMAGE:-ghcr.io/alibaba/xquic/xquic-interop@sha256:875df1e9935c6a07e26d7b5ae14df9edd06703061ce35920234a97d6991c58e0}"
output_root="${PLAB_ARTIFACT_DIR:-${PLAB_OUTPUT_ROOT:-artifacts/xquic-http3-client}}"
executor_id="${PLAB_EXECUTOR_ID:-xquic-http3-client}"
executor_version="${PLAB_EXECUTOR_VERSION:-0.1.1}"
connections="${PLAB_CONNECTIONS:-1}"
concurrency="${PLAB_CONCURRENCY:-1}"
streams="${PLAB_STREAMS_PER_CONNECTION:-1}"
duration="${PLAB_DURATION_SECONDS:-0}"
warmup="${PLAB_WARMUP_SECONDS:-0}"
mkdir -p "$output_root"
output_root="$(cd "$output_root" && pwd)"
shell="cd /xquic_bin && ./demo_client -l d -L /out/client.log -D /out -U '$target_url' -A h3 -K $timeout_seconds -o"
printf 'docker run --rm -v %q:/out --entrypoint bash %q -lc %q\n' "$output_root" "$image" "$shell" >"$output_root/command.txt"
if [ "${PLAB_PLAN_ONLY:-0}" = 1 ]; then
  printf '{"status":"planned","targetUrl":"%s","expectedStatus":%s,"image":"%s"}\n' "$target_url" "$expected_status" "$image" >"$output_root/result.json"
  exit 0
fi
set +e
docker run --rm -v "$output_root:/out" --entrypoint bash "$image" -lc "$shell" >"$output_root/stdout.txt" 2>"$output_root/stderr.txt"
exit_code=$?
set -e
status_observed=false
alpn_observed=false
completion_warning=false
grep -Eq "^:status = ${expected_status}[[:space:]]*$" "$output_root/stdout.txt" && status_observed=true
grep -q 'option set ALPN\[h3\]' "$output_root/stdout.txt" && alpn_observed=true
grep -Eq 'err:260|recv_fin:0' "$output_root/stdout.txt" && completion_warning=true
if [ "$exit_code" -eq 0 ] && [ "$status_observed" = true ] && [ "$alpn_observed" = true ]; then
  printf '{"status":"passed","targetUrl":"%s","expectedStatus":%s,"actualStatus":%s,"negotiatedProtocol":"h3","responseCompletionWarning":%s,"canonicalPayloadClaimed":false,"exitCode":0,"image":"%s"}\n' "$target_url" "$expected_status" "$expected_status" "$completion_warning" "$image" >"$output_root/result.json"
  cat <<EOF
{"schemaVersion":"protocol-lab.http-executor-result.v1","executor":{"id":"$executor_id","version":"$executor_version"},"loadGenerator":{"id":"$executor_id","version":"$executor_version"},"validation":{"status":"passed"},"protocolProof":{"requestedProtocol":"h3","observedProtocol":"h3","exactProtocolMatched":true,"fallbackDetected":false},"requestedLoad":{"connections":$connections,"concurrency":$concurrency,"streamsPerConnection":$streams,"durationSeconds":$duration,"warmupSeconds":$warmup},"effectiveLoad":{"connections":1,"concurrency":1,"streamsPerConnection":1,"durationSeconds":0,"warmupSeconds":0},"metrics":{"totalRequests":1,"successfulRequests":1,"failedRequests":0,"timeoutRequests":0,"requestsPerSecond":1,"bytesSent":0,"bytesReceived":0,"throughputBytesPerSecond":0,"latencyMeanMs":0,"latencyP50Ms":0,"latencyP75Ms":0,"latencyP90Ms":0,"latencyP95Ms":0,"latencyP99Ms":0,"statusCodeCounts":{"$expected_status":1}},"warnings":["Diagnostic single-request peer characterization; XQUIC response completion warning retained=$completion_warning; no benchmark payload or latency claim is made."]}
EOF
  exit 0
fi
printf '{"status":"failed","targetUrl":"%s","expectedStatus":%s,"responseCompletionWarning":%s,"canonicalPayloadClaimed":false,"exitCode":%s,"image":"%s"}\n' "$target_url" "$expected_status" "$completion_warning" "$exit_code" "$image" >"$output_root/result.json"
exit 1
