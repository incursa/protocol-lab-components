#!/usr/bin/env bash
set -euo pipefail
image="${PLAB_WEBSOCAT_HTTP1_WEBSOCKET_IMAGE:-incursa-protocol-lab-websocat-http1-websocket:0.1.0}"
port="${PLAB_TARGET_PORT:-18082}"
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
artifact_root="${PLAB_ARTIFACT_DIR:-$script_dir/artifacts/websocat-http1-websocket}"
mkdir -p "$artifact_root"
cd "$script_dir"
if [[ "${PLAB_SKIP_BUILD:-false}" != "true" ]]; then docker build --pull -f docker/Websocat.Dockerfile -t "$image" docker; fi
version="$(docker run --rm --entrypoint /usr/local/bin/websocat "$image" --version)"
[[ "$version" == *"1.14.1"* ]] || { echo "websocat version proof failed: $version" >&2; exit 1; }
printf '%s\n' "$version" > "$artifact_root/version.txt"
if [[ "${PLAB_PROOF_ONLY:-false}" == "true" ]]; then exit 0; fi
exec docker run --rm -p "$port:18081/tcp" "$image"
