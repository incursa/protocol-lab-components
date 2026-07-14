#!/usr/bin/env bash
set -euo pipefail
image="${PLAB_SECURE_DNS_IMAGE:-incursa-protocol-lab-bind9-dot:0.1.0}";port="${PLAB_SECURE_DNS_PORT:-20530}";cd "$(dirname "${BASH_SOURCE[0]}")"
[[ "${PLAB_SKIP_BUILD:-false}" == true ]] || docker build --pull -t "$image" docker
if [[ "${PLAB_PROOF_ONLY:-false}" == true ]]; then docker run --rm --entrypoint named "$image" -V; exit 0; fi
exec docker run --rm -p "$port:853/tcp" "$image"

