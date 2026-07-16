#!/usr/bin/env sh
set -eu
/usr/local/bin/plab-knot-resolver-fixture \
  -authority 127.0.0.1:5353 \
  -listen 0.0.0.0:444 \
  -knot-config /opt/protocol-lab/knot-resolver/config.yaml &
exec /usr/bin/knot-resolver -c /opt/protocol-lab/knot-resolver/config.yaml
