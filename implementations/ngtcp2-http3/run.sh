#!/usr/bin/env bash
set -euo pipefail

mode="${PLAB_MODE:-Client}"
image="${HTTP3_NGTCP2_IMAGE:-incursa-protocol-lab-ngtcp2-http3:0.1.2}"
artifact_root="${PLAB_ARTIFACT_ROOT:-artifacts/ngtcp2-http3}"
plan_only="${PLAB_PLAN_ONLY:-0}"
mkdir -p "$artifact_root"

if [ "$mode" = "Server" ]; then
  port="${PLAB_PORT:-4433}"
  command="docker run --rm -p ${port}:4433/udp -v $PWD/$artifact_root:/logs --entrypoint /bin/sh $image -lc 'mkdir -p /tmp/www/bytes /tmp/certs /logs/qlog /logs/sslkeylog && perl -e '\''binmode STDOUT; for ($i = 0; $i < 1024; $i++) { print chr($i % 251) }'\'' > /tmp/www/bytes/1024 && perl -e '\''binmode STDOUT; for ($i = 0; $i < 65536; $i++) { print chr($i % 251) }'\'' > /tmp/www/bytes/65536 && perl -e '\''binmode STDOUT; for ($i = 0; $i < 1048576; $i++) { print chr($i % 251) }'\'' > /tmp/www/bytes/1048576 && printf '\''%s\n'\'' '\''application/octet-stream 1024 65536 1048576'\'' > /tmp/mime.types && printf '\''%s\n'\'' '\''{\"protocol\":\"h3\",\"server\":\"ngtcp2-nghttp3\",\"implementation\":\"ngtcp2-http3\",\"utc\":\"static\",\"processId\":1}'\'' > /tmp/www/status && openssl req -x509 -newkey rsa:2048 -nodes -subj /CN=localhost -days 3650 -keyout /tmp/certs/priv.key -out /tmp/certs/cert.pem >/tmp/certs/openssl.out 2>/tmp/certs/openssl.err && /usr/local/bin/wsslserver --htdocs=/tmp/www --mime-types-file=/tmp/mime.types --qlog-dir=/logs/qlog --no-http-dump --timeout=15s --handshake-timeout=10s 0.0.0.0 4433 /tmp/certs/priv.key /tmp/certs/cert.pem'"
else
  url="${PLAB_URL:-https://host.docker.internal:4433/bytes/1024}"
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
