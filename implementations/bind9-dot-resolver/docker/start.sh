#!/usr/bin/env sh
set -eu
named -g -c /etc/bind/plab/authority.conf &
/usr/local/bin/plab-bind-control -listen 0.0.0.0:854 -rndc-config /etc/bind/plab/rndc.conf &
exec named -g -c /etc/bind/plab/resolver.conf
