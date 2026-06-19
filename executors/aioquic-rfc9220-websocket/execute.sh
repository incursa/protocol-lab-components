#!/usr/bin/env bash
set -euo pipefail

target_url="${PLAB_TARGET_URL:-https://host.docker.internal:4435/websocket-proof}"
timeout_seconds="${PLAB_TIMEOUT_SECONDS:-20}"
image="${AIOQUIC_RFC9220_WEBSOCKET_IMAGE:-incursa-protocol-lab-aioquic-rfc9220-websocket:0.1.0}"
output_root="${PLAB_OUTPUT_ROOT:-artifacts/aioquic-rfc9220-websocket}"
skip_build="${PLAB_SKIP_BUILD:-0}"
plan_only="${PLAB_PLAN_ONLY:-0}"
docker_network="${PLAB_DOCKER_NETWORK:-}"

mkdir -p "$output_root/qlog" "$output_root/sslkeylog"
command_file="$output_root/command.txt"
result_file="$output_root/result.json"

build_command=(docker build --build-arg AIOQUIC_VERSION=1.3.0 -f docker/aioquic-rfc9220-websocket.Dockerfile -t "$image" .)
run_command=(
  docker run --rm
  --add-host=host.docker.internal:host-gateway
)
if [ -n "$docker_network" ]; then
  run_command+=(--network "$docker_network")
fi
run_command+=(
  -v "$PWD/$output_root:/proof"
  -e QLOGDIR=/proof/qlog
  -e SSLKEYLOGFILE=/proof/sslkeylog/keys.log
  "$image"
  /usr/local/bin/aioquic-http3-websocket-client
  "$target_url"
  /proof/client-result.json
  --timeout "$timeout_seconds"
)

: > "$command_file"
if [ "$skip_build" != "1" ]; then
  printf '%q ' "${build_command[@]}" >> "$command_file"
  printf '\n' >> "$command_file"
fi
printf '%q ' "${run_command[@]}" >> "$command_file"
printf '\n' >> "$command_file"

if [ "$plan_only" = "1" ]; then
  cat > "$result_file" <<EOF
{"status":"planned","targetUrl":"$target_url","image":"$image"}
EOF
  echo "Planned aioquic RFC9220 WebSocket executor command at $command_file"
  exit 0
fi

if [ "$skip_build" != "1" ]; then
  "${build_command[@]}"
fi

set +e
"${run_command[@]}" > "$output_root/stdout.txt" 2> "$output_root/stderr.txt"
exit_code=$?
set -e

client_status="missing"
if [ -f "$output_root/client-result.json" ]; then
  client_status="$(python -c 'import json,sys; print(json.load(open(sys.argv[1])).get("status", "missing"))' "$output_root/client-result.json")"
fi

if [ "$exit_code" -eq 0 ] && [ "$client_status" = "passed" ]; then
  cat > "$result_file" <<EOF
{"status":"passed","targetUrl":"$target_url","image":"$image","exitCode":$exit_code}
EOF
  echo "aioquic RFC9220 WebSocket executor passed for $target_url"
  exit 0
fi

cat > "$result_file" <<EOF
{"status":"failed","targetUrl":"$target_url","image":"$image","exitCode":$exit_code,"clientStatus":"$client_status"}
EOF
exit 1
