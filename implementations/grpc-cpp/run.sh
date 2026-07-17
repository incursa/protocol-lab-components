#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"; image="${PLAB_IMAGE:-incursa-protocol-lab-grpc-cpp:0.1.2}"; port="${PLAB_TARGET_PORT:-18444}"
if [[ "${PLAB_SKIP_BUILD:-false}" != true ]]; then docker build --pull -f docker/Dockerfile -t "$image" .; fi
if [[ "${PLAB_PROOF_ONLY:-false}" == true ]]; then exec docker run --rm --entrypoint dpkg-query "$image" -W libgrpc++1.51 libprotobuf32; fi
exec docker run --rm -p "$port:18444/tcp" "$image"
