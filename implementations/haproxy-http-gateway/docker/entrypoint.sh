#!/bin/sh
set -eu

/bin/busybox httpd -f -p 127.0.0.1:9000 -h /srv -c /etc/httpd.conf &
fixture_pid=$!

cleanup() {
    kill "$fixture_pid" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

exec /usr/local/bin/docker-entrypoint.sh haproxy -W -db -f /usr/local/etc/haproxy/haproxy.cfg
