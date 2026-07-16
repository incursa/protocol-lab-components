#!/usr/bin/env bash
set -euo pipefail

image="${HTTP3_NEQO_IMAGE:-incursa-protocol-lab-neqo-http3:0.1.1}"
port="${PLAB_PORT:-4433}"
artifact_root="${PLAB_ARTIFACT_ROOT:-artifacts/neqo-http3}"
mkdir -p "$artifact_root"
shell='set -euo pipefail; export PATH=/neqo/bin:$PATH; export LD_LIBRARY_PATH=/neqo/lib:${LD_LIBRARY_PATH:-}; db=/tmp/neqo-db; mkdir -p "$db" /tmp/certs /logs; /neqo/bin/certutil -N -d "sql:$db" --empty-password; openssl req -x509 -newkey rsa:2048 -nodes -subj /CN=localhost -days 3650 -keyout /tmp/certs/priv.key -out /tmp/certs/cert.pem >/logs/openssl.stdout 2>/logs/openssl.stderr; p12=$(mktemp); openssl pkcs12 -export -in /tmp/certs/cert.pem -inkey /tmp/certs/priv.key -name cert -passout pass: -out "$p12"; /neqo/bin/pk12util -d "sql:$db" -i "$p12" -W ""; exec /neqo/bin/neqo-server -d "$db" -k cert 0.0.0.0:4433'
printf 'docker run --rm -p %s:4433/udp --entrypoint bash %s -lc %q\n' "$port" "$image" "$shell" >"$artifact_root/command.txt"
if [ "${PLAB_PLAN_ONLY:-0}" = 1 ]; then
  printf '{"status":"planned","image":"%s","port":%s}\n' "$image" "$port" >"$artifact_root/result.json"
  exit 0
fi
docker run --rm -p "$port:4433/udp" --entrypoint bash "$image" -lc "$shell" >"$artifact_root/stdout.txt" 2>"$artifact_root/stderr.txt"
printf '{"status":"passed","image":"%s","exitCode":0}\n' "$image" >"$artifact_root/result.json"
