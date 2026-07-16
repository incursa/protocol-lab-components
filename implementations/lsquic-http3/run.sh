#!/usr/bin/env bash
set -euo pipefail

image="${HTTP3_LSQUIC_IMAGE:-incursa-protocol-lab-lsquic-http3:0.1.0}"
port="${PLAB_PORT:-4433}"
artifact_root="${PLAB_ARTIFACT_ROOT:-artifacts/lsquic-http3}"
mkdir -p "$artifact_root"
shell="set -euo pipefail; mkdir -p /tmp/www /tmp/certs /logs; printf 'lsquic-http3-peer\\n' >/tmp/www/index.html; openssl req -x509 -newkey rsa:2048 -nodes -subj /CN=localhost -days 3650 -keyout /tmp/certs/priv.key -out /tmp/certs/cert.pem >/logs/openssl.stdout 2>/logs/openssl.stderr; exec /usr/bin/http_server -o version=h3 -c localhost,/tmp/certs/cert.pem,/tmp/certs/priv.key -c host.docker.internal,/tmp/certs/cert.pem,/tmp/certs/priv.key -c plab-worker-sut-01,/tmp/certs/cert.pem,/tmp/certs/priv.key -c plab-worker-load-01,/tmp/certs/cert.pem,/tmp/certs/priv.key -c 10.50.0.11,/tmp/certs/cert.pem,/tmp/certs/priv.key -c 10.50.0.12,/tmp/certs/cert.pem,/tmp/certs/priv.key -s 0.0.0.0:4433 -r /tmp/www -L info"
printf 'docker run --rm -p %s:4433/udp --entrypoint bash %s -lc %q\n' "$port" "$image" "$shell" >"$artifact_root/command.txt"
if [ "${PLAB_PLAN_ONLY:-0}" = 1 ]; then
  printf '{"status":"planned","image":"%s","port":%s}\n' "$image" "$port" >"$artifact_root/result.json"
  exit 0
fi
docker run --rm -p "$port:4433/udp" --entrypoint bash "$image" -lc "$shell" >"$artifact_root/stdout.txt" 2>"$artifact_root/stderr.txt"
printf '{"status":"passed","image":"%s","exitCode":0}\n' "$image" >"$artifact_root/result.json"
