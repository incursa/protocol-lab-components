#!/usr/bin/env sh
set -eu
setpriv --reuid=_unbound --regid=_unbound --init-groups /usr/local/bin/plab-unbound-fixture \
  -authority 127.0.0.1:5353 \
  -listen 0.0.0.0:444 \
  -unbound-config /opt/unbound/etc/unbound/plab/unbound.conf &
exec /opt/unbound/sbin/unbound -d -c /opt/unbound/etc/unbound/plab/unbound.conf
