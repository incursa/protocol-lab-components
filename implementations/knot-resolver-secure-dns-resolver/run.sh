#!/usr/bin/env bash
set -euo pipefail
image="${PLAB_SECURE_DNS_IMAGE:-incursa-protocol-lab-knot-resolver-secure-dns-resolver:0.1.2}"
dot_port="${PLAB_SECURE_DNS_PORT:-20566}"
doh2_port="${PLAB_DOH2_PORT:-$((dot_port + 1))}"
control_port="${PLAB_RESOLVER_CONTROL_PORT:-$((dot_port + 2))}"
cd "$(dirname "$0")"
if [[ "${PLAB_SKIP_BUILD:-false}" != "true" ]]; then docker build --pull -t "$image" docker; fi
if [[ "${PLAB_PROOF_ONLY:-false}" == "true" ]]; then exec docker run --rm --entrypoint /usr/bin/knot-resolver "$image" --version; fi
exec docker run --rm -p "${dot_port}:853/tcp" -p "${doh2_port}:443/tcp" -p "${control_port}:444/tcp" "$image"
