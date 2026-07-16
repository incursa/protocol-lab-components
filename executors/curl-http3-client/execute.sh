#!/usr/bin/env bash
set -euo pipefail

target_url="${PLAB_TARGET_BASE_URL:-${PLAB_TARGET_URL:-https://host.docker.internal:8443/status}}"
expected_status="${PLAB_EXPECTED_STATUS:-200}"
timeout_seconds="${PLAB_TIMEOUT_SECONDS:-15}"
image="${HTTP3_CURL_IMAGE:-ghcr.io/macbre/curl-http3}"
output_root="${PLAB_ARTIFACT_DIR:-${PLAB_OUTPUT_ROOT:-artifacts/curl-http3-client}}"
plan_only="${PLAB_PLAN_ONLY:-0}"
executor_id="${PLAB_EXECUTOR_ID:-curl-http3-client}"
executor_version="${PLAB_EXECUTOR_VERSION:-0.1.7}"
connections="${PLAB_CONNECTIONS:-1}"
concurrency="${PLAB_CONCURRENCY:-1}"
streams="${PLAB_STREAMS_PER_CONNECTION:-1}"
duration="${PLAB_DURATION_SECONDS:-0}"
warmup="${PLAB_WARMUP_SECONDS:-0}"
container_target_url="${target_url/localhost/host.docker.internal}"
container_target_url="${container_target_url/127.0.0.1/host.docker.internal}"
docker_host_args=()
if [ "$(uname -s)" = "Linux" ]; then
  docker_host_args+=(--add-host=host.docker.internal:host-gateway)
fi

mkdir -p "$output_root"
output_root="$(cd "$output_root" && pwd)"
command_file="$output_root/command.txt"
result_file="$output_root/result.json"

cat > "$command_file" <<EOF
docker run --rm ${docker_host_args[*]} -v "$output_root:/out" "$image" curl --http3-only --max-time "$timeout_seconds" --insecure --silent --show-error --output /out/body.bin --write-out '%{http_code}' "$container_target_url"
EOF

if [ "$plan_only" = "1" ]; then
  cat > "$result_file" <<EOF
{"status":"planned","targetUrl":"$target_url","expectedStatus":$expected_status,"image":"$image"}
EOF
  echo "Planned curl HTTP/3 executor command at $command_file"
  exit 0
fi

status="$(docker run --rm "${docker_host_args[@]}" -v "$output_root:/out" "$image" curl --http3-only --max-time "$timeout_seconds" --insecure --silent --show-error --output /out/body.bin --write-out '%{http_code}' "$container_target_url" > "$output_root/stdout.txt" 2> "$output_root/stderr.txt" && cat "$output_root/stdout.txt")"
if [ "$status" != "$expected_status" ]; then
  cat > "$result_file" <<EOF
{"status":"failed","targetUrl":"$target_url","expectedStatus":$expected_status,"actualStatus":"$status","image":"$image"}
EOF
  exit 1
fi

cat > "$result_file" <<EOF
{"status":"passed","targetUrl":"$target_url","expectedStatus":$expected_status,"actualStatus":"$status","image":"$image"}
EOF
bytes_received="$(wc -c < "$output_root/body.bin" | tr -d '[:space:]')"
cat <<EOF
{"schemaVersion":"protocol-lab.http-executor-result.v1","executor":{"id":"$executor_id","version":"$executor_version"},"loadGenerator":{"id":"$executor_id","version":"$executor_version"},"validation":{"status":"passed"},"protocolProof":{"requestedProtocol":"h3","observedProtocol":"h3","exactProtocolMatched":true,"fallbackDetected":false},"requestedLoad":{"connections":$connections,"concurrency":$concurrency,"streamsPerConnection":$streams,"durationSeconds":$duration,"warmupSeconds":$warmup},"effectiveLoad":{"connections":1,"concurrency":1,"streamsPerConnection":1,"durationSeconds":0,"warmupSeconds":0},"metrics":{"totalRequests":1,"successfulRequests":1,"failedRequests":0,"timeoutRequests":0,"requestsPerSecond":1,"bytesSent":0,"bytesReceived":$bytes_received,"throughputBytesPerSecond":0,"latencyMeanMs":0,"latencyP50Ms":0,"latencyP75Ms":0,"latencyP90Ms":0,"latencyP95Ms":0,"latencyP99Ms":0,"statusCodeCounts":{"$status":1}},"warnings":["Diagnostic single-request peer characterization; no benchmark payload or latency claim is made."]}
EOF
