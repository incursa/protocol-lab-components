#!/usr/bin/env bash
set -euo pipefail
image="${PLAB_SECURE_DNS_IMAGE:-incursa-protocol-lab-unbound-doh2-resolver:0.1.0}"
port="${PLAB_SECURE_DNS_PORT:-20564}"
control_port="${PLAB_RESOLVER_CONTROL_PORT:-$((port + 1))}"
cd "$(dirname "$0")"
if [[ "${PLAB_SKIP_BUILD:-false}" != "true" ]]; then docker build --pull -t "$image" docker; fi
if [[ "${PLAB_PROOF_ONLY:-false}" == "true" ]]; then exec docker run --rm --entrypoint /opt/unbound/sbin/unbound "$image" -V; fi
exec docker run --rm -p "${port}:443/tcp" -p "${control_port}:444/tcp" "$image"
