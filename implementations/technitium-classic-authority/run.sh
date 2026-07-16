#!/usr/bin/env bash
set -euo pipefail
image="${PLAB_DNS_CLASSIC_IMAGE:-incursa-protocol-lab-technitium-classic-authority:0.1.1}"
port="${PLAB_DNS_CLASSIC_PORT:-15355}"
cd "$(dirname "${BASH_SOURCE[0]}")"
[[ "${PLAB_SKIP_BUILD:-false}" == true ]] || docker build --pull -t "$image" docker
if [[ "${PLAB_PROOF_ONLY:-false}" == true ]]; then
  docker run --rm -e PLAB_PLAN_ONLY=true "$image"
  exit 0
fi
exec docker run --rm -p "$port:53/udp" -p "$port:53/tcp" "$image"
