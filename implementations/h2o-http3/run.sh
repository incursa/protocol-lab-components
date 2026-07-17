#!/usr/bin/env bash
set -euo pipefail

image="${PLAB_IMAGE:-incursa-protocol-lab-h2o-http3:0.1.1}"
port="${PLAB_HTTP_PORT:-5448}"
skip_build="${PLAB_SKIP_BUILD:-0}"
plan_only="${PLAB_PLAN_ONLY:-0}"
proof_only="${PLAB_PROOF_ONLY:-0}"
artifact_root="${PLAB_ARTIFACT_ROOT:-artifacts/h2o-http3}"

mkdir -p "$artifact_root"
commands=()
if [[ "$skip_build" != "1" ]]; then
  commands+=("docker build --pull -f docker/H2O.Dockerfile -t $image docker")
fi
commands+=("docker run --rm --entrypoint h2o $image -v")
commands+=("docker run --rm -p ${port}:8443/tcp -p ${port}:8443/udp $image")
printf '%s\n' "${commands[@]}" > "$artifact_root/command.txt"

if [[ "$plan_only" = "1" ]]; then
  printf '{"status":"planned","image":"%s","port":%s}\n' "$image" "$port" > "$artifact_root/result.json"
  exit 0
fi

if [[ "$skip_build" != "1" ]]; then
  docker build --pull -f docker/H2O.Dockerfile -t "$image" docker
fi

version_output="$(docker run --rm --entrypoint h2o "$image" -v 2>&1)"
printf '%s\n' "$version_output" > "$artifact_root/h2o-version.txt"
if [[ "$version_output" != *h2o* ]]; then
  printf '%s\n' "Selected image '$image' did not identify the H2O server." >&2
  exit 78
fi

if [[ "$proof_only" = "1" ]]; then
  printf '{"status":"proved","image":"%s","versionPath":"%s"}\n' "$image" "$artifact_root/h2o-version.txt" > "$artifact_root/result.json"
  exit 0
fi

docker run --rm -p "${port}:8443/tcp" -p "${port}:8443/udp" "$image"
