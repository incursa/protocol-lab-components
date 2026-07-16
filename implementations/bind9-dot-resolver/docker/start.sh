#!/usr/bin/env sh
set -eu
named -g -c /opt/protocol-lab/bind-resolver/authority.conf &
/usr/local/bin/plab-bind-control -listen 0.0.0.0:854 -rndc-config /opt/protocol-lab/bind-resolver/rndc.conf &
exec named -g -c /opt/protocol-lab/bind-resolver/resolver.conf
