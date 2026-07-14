#!/usr/bin/env bash
set -euo pipefail

image="${PLAB_NGINX_HTTP2_IMAGE:-incursa-protocol-lab-nginx-http2:0.1.0}"
port="${PLAB_HTTP_PORT:-${PORT:-8084}}"
skip_build="${PLAB_SKIP_BUILD:-false}"
plan_only="${PLAB_PLAN_ONLY:-false}"
proof_only="${PLAB_PROOF_ONLY:-false}"
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
artifact_root="${PLAB_ARTIFACT_DIR:-$script_dir/artifacts/nginx-http2}"
mkdir -p "$artifact_root"

commands=()
if [[ "$skip_build" != "true" ]]; then
  commands+=("docker build --pull -f docker/nginx.Dockerfile -t $image docker")
fi
commands+=("docker run --rm --entrypoint nginx $image -V")
commands+=("docker run --rm $image nginx -t")
commands+=("docker run --rm -p $port:8080/tcp $image")
printf '%s\n' "${commands[@]}" > "$artifact_root/command.txt"

if [[ "$plan_only" == "true" ]]; then
  printf '{"status":"planned","image":"%s","port":%s,"protocolVariant":"h2c-prior-knowledge"}\n' "$image" "$port" > "$artifact_root/result.json"
  exit 0
fi

cd "$script_dir"
if [[ "$skip_build" != "true" ]]; then
  docker build --pull -f docker/nginx.Dockerfile -t "$image" docker
fi
version="$(docker run --rm --entrypoint nginx "$image" -V 2>&1)"
printf '%s\n' "$version" > "$artifact_root/nginx-version.txt"
[[ "$version" == *--with-http_v2_module* ]] || { echo "Selected image lacks --with-http_v2_module" >&2; exit 1; }
docker run --rm "$image" nginx -t
if [[ "$proof_only" == "true" ]]; then
  printf '{"status":"proved","image":"%s","protocolVariant":"h2c-prior-knowledge"}\n' "$image" > "$artifact_root/result.json"
  exit 0
fi
exec docker run --rm -p "$port:8080/tcp" "$image"
