#!/bin/sh
set -eu
if [ "${1:-}" = "--version" ]; then
  printf 'envoy-connect-udp 0.1.0 envoy v1.38.3\n'
  exit 0
fi
/usr/local/bin/udp-echo &
echo_pid=$!
trap 'kill "$echo_pid" 2>/dev/null || true' EXIT INT TERM
printf 'role=proxy implementation=envoy version=v1.38.3 authority=masque-proxy.plab.test bind=:4443 protocol=connect-udp upstream_status=alpha ready=starting\n'
exec /usr/local/bin/envoy -c /etc/envoy/envoy.yaml --concurrency 1 --disable-hot-restart --log-level info
