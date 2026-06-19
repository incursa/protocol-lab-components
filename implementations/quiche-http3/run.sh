#!/usr/bin/env bash
set -euo pipefail

mode="${PLAB_MODE:-Client}"
image="${HTTP3_QUICHE_IMAGE:-cloudflare/quiche:latest}"
artifact_root="${PLAB_ARTIFACT_ROOT:-artifacts/quiche-http3}"
plan_only="${PLAB_PLAN_ONLY:-0}"
mkdir -p "$artifact_root"

if [ "$mode" = "Server" ]; then
  port="${PLAB_PORT:-4433}"
  www_root="${PLAB_WWW_ROOT:-www}"
  cert_path="${PLAB_CERT_PATH:-certs/cert.pem}"
  key_path="${PLAB_KEY_PATH:-certs/priv.key}"
  command="docker run --rm -p ${port}:4433/udp -v $PWD/$www_root:/www:ro -v $PWD/$(dirname "$cert_path"):/certs:ro --entrypoint /bin/sh $image -lc 'mkdir -p /logs/qlog /logs/sslkeylog && quiche-server --listen 0.0.0.0:4433 --cert /certs/$(basename "$cert_path") --key /certs/$(basename "$key_path") --root /www --http-version HTTP/3'"
else
  url="${PLAB_URL:-https://host.docker.internal:4433/small.txt}"
  connect_to="${PLAB_CONNECT_TO:-host.docker.internal:4433}"
  command="docker run --rm -v $PWD/$artifact_root:/downloads --entrypoint /bin/sh $image -lc 'mkdir -p /downloads /logs/qlog /logs/sslkeylog && quiche-client --http-version HTTP/3 --no-verify --connect-to $connect_to --dump-responses /downloads --dump-json --max-json-payload 0 $url'"
fi

printf '%s\n' "$command" > "$artifact_root/command.txt"
if [ "$plan_only" = "1" ]; then
  echo "{\"status\":\"planned\",\"mode\":\"$mode\",\"image\":\"$image\"}" > "$artifact_root/result.json"
  echo "Planned quiche HTTP/3 $mode command at $artifact_root/command.txt"
  exit 0
fi

eval "$command" > "$artifact_root/stdout.txt" 2> "$artifact_root/stderr.txt"
