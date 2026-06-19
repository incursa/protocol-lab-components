#!/usr/bin/env bash
set -euo pipefail

mode="${PLAB_MODE:-Client}"
image="${HTTP3_NGTCP2_IMAGE:-ghcr.io/ngtcp2/ngtcp2-interop:latest}"
artifact_root="${PLAB_ARTIFACT_ROOT:-artifacts/ngtcp2-http3}"
plan_only="${PLAB_PLAN_ONLY:-0}"
mkdir -p "$artifact_root"

if [ "$mode" = "Server" ]; then
  port="${PLAB_PORT:-4433}"
  www_root="${PLAB_WWW_ROOT:-www}"
  cert_path="${PLAB_CERT_PATH:-certs/cert.pem}"
  key_path="${PLAB_KEY_PATH:-certs/priv.key}"
  command="docker run --rm -p ${port}:4433/udp -v $PWD/$www_root:/www:ro -v $PWD/$(dirname "$cert_path"):/certs:ro -v $PWD/$artifact_root:/logs --entrypoint /bin/sh $image -lc 'mkdir -p /logs/qlog /logs/sslkeylog && /usr/local/bin/wsslserver --htdocs=/www --qlog-dir=/logs/qlog --no-http-dump --timeout=15s --handshake-timeout=10s 0.0.0.0 4433 /certs/$(basename "$key_path") /certs/$(basename "$cert_path")'"
else
  url="${PLAB_URL:-https://host.docker.internal:4433/small.txt}"
  host_name="${PLAB_HOST_NAME:-host.docker.internal}"
  peer_port="${PLAB_PEER_PORT:-4433}"
  command="docker run --rm -v $PWD/$artifact_root:/downloads -v $PWD/$artifact_root:/logs --entrypoint /bin/sh $image -lc 'mkdir -p /downloads /logs/qlog /logs/sslkeylog && wsslclient --download=/downloads --exit-on-all-streams-close --timeout=15s --handshake-timeout=10s --no-http-dump --qlog-dir=/logs/qlog $host_name $peer_port $url'"
fi

printf '%s\n' "$command" > "$artifact_root/command.txt"
if [ "$plan_only" = "1" ]; then
  echo "{\"status\":\"planned\",\"mode\":\"$mode\",\"image\":\"$image\"}" > "$artifact_root/result.json"
  echo "Planned ngtcp2/nghttp3 HTTP/3 $mode command at $artifact_root/command.txt"
  exit 0
fi

eval "$command" > "$artifact_root/stdout.txt" 2> "$artifact_root/stderr.txt"
