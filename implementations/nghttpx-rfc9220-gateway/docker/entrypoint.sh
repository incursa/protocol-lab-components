#!/usr/bin/env sh
set -eu

python3 -u /usr/local/lib/protocol-lab/backend.py &
backend_pid=$!

terminate() {
  kill "$proxy_pid" "$backend_pid" 2>/dev/null || true
}
trap terminate INT TERM EXIT

nghttpx \
  --frontend='0.0.0.0,4433;quic' \
  --backend='127.0.0.1,8080' \
  --workers=1 \
  --frontend-http3-idle-timeout=30s \
  --backend-read-timeout=30s \
  --backend-write-timeout=30s \
  --accesslog-file=/dev/null \
  --errorlog-file=/dev/stderr \
  /certs/leaf-key.pem /certs/leaf.pem &
proxy_pid=$!

wait "$proxy_pid"
