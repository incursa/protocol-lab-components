#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
image="${PLAB_IMAGE:-incursa-protocol-lab-grpc-dotnet:0.1.0}"; port="${PLAB_TARGET_PORT:-18444}"
if [[ "${PLAB_SKIP_BUILD:-false}" != true ]]; then docker build --pull -f docker/Dockerfile -t "$image" .; fi
if [[ "${PLAB_PROOF_ONLY:-false}" == true ]]; then exec docker run --rm --entrypoint dotnet "$image" --info; fi
exec docker run --rm -p "$port:18444/tcp" "$image"
