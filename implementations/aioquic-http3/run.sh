#!/usr/bin/env bash
set -euo pipefail

mode="${PLAB_MODE:-Client}"
image="${PLAB_IMAGE:-incursa-protocol-lab-aioquic-http3:0.3.0}"
plan_only="${PLAB_PLAN_ONLY:-0}"
skip_build="${PLAB_SKIP_BUILD:-0}"
artifact_root="${PLAB_ARTIFACT_ROOT:-artifacts/aioquic-http3}"
mkdir -p "$artifact_root"

commands=()
if [ "$skip_build" != "1" ]; then
  commands+=("docker build --build-arg AIOQUIC_VERSION=1.3.0 -f docker/aioquic.Dockerfile -t $image .")
fi

if [ "$mode" = "Server" ]; then
  port="${PLAB_PORT:-4433}"
  www_root="${PLAB_WWW_ROOT:-www}"
  cert_path="${PLAB_CERT_PATH:-certs/leaf.pem}"
  key_path="${PLAB_KEY_PATH:-certs/leaf-key.pem}"
  commands+=("docker run --rm -p ${port}:4433/udp -v $PWD/$www_root:/www:ro -v $PWD/$(dirname "$cert_path"):/certs:ro $image /usr/local/bin/aioquic-http3-server /www /certs/$(basename "$cert_path") /certs/$(basename "$key_path") 4433")
else
  url="${PLAB_URL:-https://host.docker.internal:8443/status}"
  expected_status="${PLAB_EXPECTED_STATUS:-200}"
  commands+=("docker run --rm -v $PWD/$artifact_root:/downloads $image /usr/local/bin/aioquic-http3-client $url /downloads/body.bin --expect-status $expected_status")
fi

printf '%s\n' "${commands[@]}" > "$artifact_root/command.txt"

if [ "$plan_only" = "1" ]; then
  echo "{\"status\":\"planned\",\"mode\":\"$mode\",\"image\":\"$image\"}" > "$artifact_root/result.json"
  echo "Planned aioquic HTTP/3 $mode command at $artifact_root/command.txt"
  exit 0
fi

if [ "$skip_build" != "1" ]; then
  docker build --build-arg AIOQUIC_VERSION=1.3.0 -f docker/aioquic.Dockerfile -t "$image" .
fi

if [ "$mode" = "Server" ]; then
  eval "${commands[-1]}"
else
  eval "${commands[-1]}" > "$artifact_root/stdout.txt" 2> "$artifact_root/stderr.txt"
fi
