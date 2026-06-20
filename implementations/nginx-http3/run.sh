#!/usr/bin/env bash
set -euo pipefail

image="${PLAB_IMAGE:-incursa-protocol-lab-nginx-http3:0.1.1}"
port="${PLAB_HTTP_PORT:-5446}"
skip_build="${PLAB_SKIP_BUILD:-0}"
plan_only="${PLAB_PLAN_ONLY:-0}"
proof_only="${PLAB_PROOF_ONLY:-0}"
artifact_root="${PLAB_ARTIFACT_ROOT:-artifacts/nginx-http3}"

mkdir -p "$artifact_root"

commands=()
if [ "$skip_build" != "1" ]; then
  commands+=("docker build --pull -f docker/nginx.Dockerfile -t $image docker")
fi
commands+=("docker run --rm --entrypoint nginx $image -V")
commands+=("docker run --rm $image nginx -t")
commands+=("docker run --rm -p ${port}:8443/tcp -p ${port}:8443/udp $image nginx -g 'daemon off;'")
printf '%s\n' "${commands[@]}" > "$artifact_root/command.txt"

if [ "$plan_only" = "1" ]; then
  printf '{"status":"planned","image":"%s","port":%s}\n' "$image" "$port" > "$artifact_root/result.json"
  echo "Planned nginx HTTP/3 command at $artifact_root/command.txt"
  exit 0
fi

if [ "$skip_build" != "1" ]; then
  docker build --pull -f docker/nginx.Dockerfile -t "$image" docker
fi

version_output="$(docker run --rm --entrypoint nginx "$image" -V 2>&1)"
printf '%s\n' "$version_output" > "$artifact_root/nginx-version.txt"
case "$version_output" in
  *--with-http_v3_module*) ;;
  *)
    printf '%s\n' "Selected nginx image '$image' does not advertise HTTP/3 support." >&2
    exit 78
    ;;
esac

docker run --rm "$image" nginx -t

if [ "$proof_only" = "1" ]; then
  printf '{"status":"proved","image":"%s","nginxVersionPath":"%s","requiredModule":"--with-http_v3_module"}\n' "$image" "$artifact_root/nginx-version.txt" > "$artifact_root/result.json"
  echo "Proved nginx HTTP/3 module support at $artifact_root/nginx-version.txt"
  exit 0
fi

docker run --rm -p "${port}:8443/tcp" -p "${port}:8443/udp" "$image" nginx -g 'daemon off;'
