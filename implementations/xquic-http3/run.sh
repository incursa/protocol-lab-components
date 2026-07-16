#!/usr/bin/env bash
set -euo pipefail

image="${HTTP3_XQUIC_IMAGE:-incursa-protocol-lab-xquic-http3:0.1.1}"
port="${PLAB_PORT:-4433}"
artifact_root="${PLAB_ARTIFACT_ROOT:-artifacts/xquic-http3}"
mkdir -p "$artifact_root"
shell="set -euo pipefail; cd /xquic_bin; mkdir -p /tmp/www /tmp/certs /logs; printf 'xquic-http3-peer\\n' >/tmp/www/index.html; openssl req -x509 -newkey rsa:2048 -nodes -subj /CN=localhost -days 3650 -keyout /tmp/certs/priv.key -out /tmp/certs/cert.pem >/logs/openssl.stdout 2>/logs/openssl.stderr; cp /tmp/certs/priv.key server.key; cp /tmp/certs/cert.pem server.crt; exec ./demo_server -l d -L /logs/server.log -p 4433 -D /tmp/www -i -M"
printf 'docker run --rm -p %s:4433/udp --entrypoint bash %s -lc %q\n' "$port" "$image" "$shell" >"$artifact_root/command.txt"
if [ "${PLAB_PLAN_ONLY:-0}" = 1 ]; then
  printf '{"status":"planned","image":"%s","port":%s}\n' "$image" "$port" >"$artifact_root/result.json"
  exit 0
fi
docker run --rm -p "$port:4433/udp" --entrypoint bash "$image" -lc "$shell" >"$artifact_root/stdout.txt" 2>"$artifact_root/stderr.txt"
printf '{"status":"passed","image":"%s","exitCode":0}\n' "$image" >"$artifact_root/result.json"
