#!/usr/bin/env bash
set -euo pipefail

image="${PLAB_CADDY_HTTP2_IMAGE:-incursa-protocol-lab-caddy-http2:0.1.0}"
port="${PLAB_HTTP_PORT:-${PORT:-8083}}"
skip_build="${PLAB_SKIP_BUILD:-false}"
plan_only="${PLAB_PLAN_ONLY:-false}"
proof_only="${PLAB_PROOF_ONLY:-false}"
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
artifact_root="${PLAB_ARTIFACT_DIR:-$script_dir/artifacts/caddy-http2}"
mkdir -p "$artifact_root"

commands=()
if [[ "$skip_build" != "true" ]]; then
  commands+=("docker build --pull -f docker/Caddy.Dockerfile -t $image docker")
fi
commands+=("docker run --rm $image caddy version")
commands+=("docker run --rm $image caddy validate --config /etc/caddy/Caddyfile --adapter caddyfile")
commands+=("docker run --rm -p $port:8080/tcp $image")
printf '%s\n' "${commands[@]}" > "$artifact_root/command.txt"

if [[ "$plan_only" == "true" ]]; then
  printf '{"status":"planned","image":"%s","port":%s,"protocolVariant":"h2c-prior-knowledge"}\n' "$image" "$port" > "$artifact_root/result.json"
  exit 0
fi

cd "$script_dir"
if [[ "$skip_build" != "true" ]]; then
  docker build --pull -f docker/Caddy.Dockerfile -t "$image" docker
fi
version="$(docker run --rm "$image" caddy version)"
printf '%s\n' "$version" > "$artifact_root/caddy-version.txt"
[[ "$version" =~ ^v2\.11\.2([[:space:]]|$) ]] || { echo "Expected Caddy v2.11.2, got: $version" >&2; exit 1; }
docker run --rm "$image" caddy validate --config /etc/caddy/Caddyfile --adapter caddyfile
if [[ "$proof_only" == "true" ]]; then
  printf '{"status":"proved","image":"%s","protocolVariant":"h2c-prior-knowledge"}\n' "$image" > "$artifact_root/result.json"
  exit 0
fi
exec docker run --rm -p "$port:8080/tcp" "$image"
