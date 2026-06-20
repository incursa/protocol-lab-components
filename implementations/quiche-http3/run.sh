#!/usr/bin/env bash
set -euo pipefail

mode="${PLAB_MODE:-Client}"
image="${HTTP3_QUICHE_IMAGE:-incursa-protocol-lab-quiche-http3:0.1.3}"
artifact_root="${PLAB_ARTIFACT_ROOT:-artifacts/quiche-http3}"
plan_only="${PLAB_PLAN_ONLY:-0}"
mkdir -p "$artifact_root"

if [ "$mode" = "Server" ]; then
  port="${PLAB_PORT:-4433}"
  command="docker run --rm -p ${port}:4433/udp --entrypoint /bin/sh $image -lc 'mkdir -p /tmp/www/bytes /tmp/certs /logs/qlog /logs/sslkeylog && perl -e '\''binmode STDOUT; for ($i = 0; $i < 1024; $i++) { print chr($i % 251) }'\'' > /tmp/www/bytes/1024 && perl -e '\''binmode STDOUT; for ($i = 0; $i < 65536; $i++) { print chr($i % 251) }'\'' > /tmp/www/bytes/65536 && perl -e '\''binmode STDOUT; for ($i = 0; $i < 1048576; $i++) { print chr($i % 251) }'\'' > /tmp/www/bytes/1048576 && printf '\''%s\n'\'' '\''{\"protocol\":\"h3\",\"server\":\"quiche\",\"implementation\":\"quiche-http3\",\"utc\":\"static\",\"processId\":1}'\'' > /tmp/www/status && openssl req -x509 -newkey rsa:2048 -nodes -subj /CN=localhost -days 3650 -keyout /tmp/certs/priv.key -out /tmp/certs/cert.pem >/tmp/certs/openssl.out 2>/tmp/certs/openssl.err && quiche-server --listen 0.0.0.0:4433 --cert /tmp/certs/cert.pem --key /tmp/certs/priv.key --root /tmp/www --http-version HTTP/3'"
else
  url="${PLAB_URL:-https://host.docker.internal:4433/bytes/1024}"
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
