#!/usr/bin/env bash
set -euo pipefail

image="${PLAB_IMAGE:-incursa-protocol-lab-caddy-http3:0.1.3}"
port="${PLAB_HTTP_PORT:-5445}"
skip_build="${PLAB_SKIP_BUILD:-0}"
plan_only="${PLAB_PLAN_ONLY:-0}"
artifact_root="${PLAB_ARTIFACT_ROOT:-artifacts/caddy-http3}"

mkdir -p "$artifact_root"

commands=()
if [ "$skip_build" != "1" ]; then
  commands+=("docker build --pull -f docker/Caddy.Dockerfile -t $image docker")
fi

commands+=("docker run --rm -p ${port}:8443/tcp -p ${port}:8443/udp $image caddy run --config /etc/caddy/Caddyfile --adapter caddyfile")
printf '%s\n' "${commands[@]}" > "$artifact_root/command.txt"

if [ "$plan_only" = "1" ]; then
  printf '{"status":"planned","image":"%s","port":%s}\n' "$image" "$port" > "$artifact_root/result.json"
  echo "Planned Caddy HTTP/3 command at $artifact_root/command.txt"
  exit 0
fi

if [ "$skip_build" != "1" ]; then
  docker build --pull -f docker/Caddy.Dockerfile -t "$image" docker
fi

docker run --rm -p "${port}:8443/tcp" -p "${port}:8443/udp" "$image" caddy run --config /etc/caddy/Caddyfile --adapter caddyfile
