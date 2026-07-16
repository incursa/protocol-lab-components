#!/usr/bin/env bash
set -euo pipefail
image="${PLAB_SECURE_DNS_IMAGE:-incursa-protocol-lab-bind9-dot-resolver:0.1.2}"
port="${PLAB_SECURE_DNS_PORT:-20562}"
control_port="${PLAB_RESOLVER_CONTROL_PORT:-$((port + 1))}"
cd "$(dirname "${BASH_SOURCE[0]}")"
[[ "${PLAB_SKIP_BUILD:-false}" == true ]] || docker build --pull -t "$image" docker
if [[ "${PLAB_PROOF_ONLY:-false}" == true ]]; then docker run --rm --entrypoint named "$image" -V; exit 0; fi
exec docker run --rm -p "$port:853/tcp" -p "$control_port:854/tcp" "$image"
