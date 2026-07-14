#!/usr/bin/env bash
set -euo pipefail
image="${PLAB_NODE_WS_WEBSOCKET_IMAGE:-incursa-protocol-lab-node-ws-websocket:0.1.0}"
port="${PLAB_TARGET_PORT:-18083}"
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
artifact_root="${PLAB_ARTIFACT_DIR:-$script_dir/artifacts/node-ws-websocket}"
mkdir -p "$artifact_root"
cd "$script_dir"
if [[ "${PLAB_SKIP_BUILD:-false}" != "true" ]]; then docker build --pull -f docker/NodeWs.Dockerfile -t "$image" docker; fi
version="$(docker run --rm "$image" --version)"
[[ "$version" == *"ws 8.21.0"* ]] || { echo "Node ws version proof failed: $version" >&2; exit 1; }
printf '%s\n' "$version" > "$artifact_root/version.txt"
if [[ "${PLAB_PROOF_ONLY:-false}" == "true" ]]; then exit 0; fi
exec docker run --rm -p "$port:18081/tcp" "$image"
