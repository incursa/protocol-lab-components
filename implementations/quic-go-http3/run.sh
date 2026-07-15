#!/usr/bin/env bash
set -euo pipefail

image="${PLAB_IMAGE:-incursa-protocol-lab-quic-go-http3:0.1.5}"
port="${PLAB_HTTP_PORT:-5446}"
skip_build="${PLAB_SKIP_BUILD:-0}"
plan_only="${PLAB_PLAN_ONLY:-0}"
artifact_root="${PLAB_ARTIFACT_ROOT:-artifacts/quic-go-http3}"

mkdir -p "$artifact_root"

commands=()
if [ "$skip_build" != "1" ]; then
  commands+=("docker build --pull --build-arg QUIC_GO_VERSION=v0.60.0 -f docker/quic-go-http3.Dockerfile -t $image .")
fi

commands+=("docker run --rm -p ${port}:4433/udp $image /quic-go-http3-server -listen :4433")
printf '%s\n' "${commands[@]}" > "$artifact_root/command.txt"

if [ "$plan_only" = "1" ]; then
  printf '{"status":"planned","image":"%s","port":%s}\n' "$image" "$port" > "$artifact_root/result.json"
  echo "Planned quic-go HTTP/3 command at $artifact_root/command.txt"
  exit 0
fi

if [ "$skip_build" != "1" ]; then
  docker build --pull --build-arg QUIC_GO_VERSION=v0.60.0 -f docker/quic-go-http3.Dockerfile -t "$image" .
fi

docker run --rm -p "${port}:4433/udp" "$image" /quic-go-http3-server -listen :4433
