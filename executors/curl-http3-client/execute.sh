#!/usr/bin/env bash
set -euo pipefail

target_url="${PLAB_TARGET_URL:-https://host.docker.internal:8443/status}"
expected_status="${PLAB_EXPECTED_STATUS:-200}"
timeout_seconds="${PLAB_TIMEOUT_SECONDS:-15}"
image="${HTTP3_CURL_IMAGE:-ghcr.io/macbre/curl-http3}"
output_root="${PLAB_OUTPUT_ROOT:-artifacts/curl-http3-client}"
plan_only="${PLAB_PLAN_ONLY:-0}"

mkdir -p "$output_root"
command_file="$output_root/command.txt"
result_file="$output_root/result.json"

cat > "$command_file" <<EOF
docker run --rm -v "$PWD/$output_root:/out" "$image" --http3-only --max-time "$timeout_seconds" --insecure --silent --show-error --output /out/body.bin --write-out '%{http_code}' "$target_url"
EOF

if [ "$plan_only" = "1" ]; then
  cat > "$result_file" <<EOF
{"status":"planned","targetUrl":"$target_url","expectedStatus":$expected_status,"image":"$image"}
EOF
  echo "Planned curl HTTP/3 executor command at $command_file"
  exit 0
fi

status="$(docker run --rm -v "$PWD/$output_root:/out" "$image" --http3-only --max-time "$timeout_seconds" --insecure --silent --show-error --output /out/body.bin --write-out '%{http_code}' "$target_url" > "$output_root/stdout.txt" 2> "$output_root/stderr.txt" && cat "$output_root/stdout.txt")"
if [ "$status" != "$expected_status" ]; then
  cat > "$result_file" <<EOF
{"status":"failed","targetUrl":"$target_url","expectedStatus":$expected_status,"actualStatus":"$status","image":"$image"}
EOF
  exit 1
fi

cat > "$result_file" <<EOF
{"status":"passed","targetUrl":"$target_url","expectedStatus":$expected_status,"actualStatus":"$status","image":"$image"}
EOF
echo "curl HTTP/3 executor passed for $target_url"
