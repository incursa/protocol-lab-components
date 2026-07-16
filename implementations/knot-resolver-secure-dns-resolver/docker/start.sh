#!/usr/bin/env sh
set -eu
config=/opt/protocol-lab/knot-resolver/config.yaml
/usr/bin/knot-resolver -c "$config" &
resolver_pid=$!
helper_pid=
terminate() {
  if [ -n "$helper_pid" ]; then kill "$helper_pid" 2>/dev/null || true; fi
  kill "$resolver_pid" 2>/dev/null || true
  wait "$resolver_pid" 2>/dev/null || true
}
trap terminate INT TERM EXIT
ready=false
attempt=0
while [ "$attempt" -lt 45 ]; do
  if /usr/bin/kresctl --config "$config" pids >/dev/null 2>&1; then ready=true; break; fi
  if ! kill -0 "$resolver_pid" 2>/dev/null; then wait "$resolver_pid"; exit $?; fi
  attempt=$((attempt + 1))
  sleep 1
done
if [ "$ready" != true ]; then echo "Knot Resolver management API did not become ready." >&2; exit 1; fi
/usr/local/bin/plab-knot-resolver-fixture \
  -authority 127.0.0.1:53 \
  -listen 0.0.0.0:444 \
  -knot-config "$config" &
helper_pid=$!
wait "$resolver_pid"
status=$?
trap - INT TERM EXIT
terminate
exit "$status"
