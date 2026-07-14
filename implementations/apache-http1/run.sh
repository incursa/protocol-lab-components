#!/usr/bin/env bash
set -euo pipefail

image='ubuntu/apache2@sha256:6563a8f98ce5469715962cf217335ec73842e56abb3720094a15f2b6747b87bc'
port="${PLAB_HTTP_PORT:-${PORT:-8080}}"
container_name="${PLAB_CONTAINER_NAME:-protocol-lab-apache-http1-$$}"
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
run_root="$(mktemp -d "${TMPDIR:-/tmp}/protocol-lab-apache-http1.XXXXXX")"
fixture_root="$run_root/fixtures"
mkdir -p "$fixture_root"
cleanup() {
  docker rm --force "$container_name" >/dev/null 2>&1 || true
  rm -rf "$run_root"
}
trap cleanup EXIT INT TERM HUP

command -v docker >/dev/null 2>&1 || { echo 'docker executable was not found on PATH.' >&2; exit 1; }
for encoded in "$script_dir"/fixtures/*.b64; do
  base64 --decode "$encoded" > "$fixture_root/$(basename "${encoded%.b64}")"
done

args=(run --rm --init --pull missing
  --publish "127.0.0.1:${port}:8080"
  --mount "type=bind,source=$fixture_root,target=/var/www/protocol-lab,readonly"
  --mount "type=bind,source=$script_dir/apache-http1.conf,target=/etc/apache2/conf-enabled/zzz-protocol-lab-http1.conf,readonly")
args+=(--name "$container_name")
args+=(--entrypoint /bin/sh "$image" -ec 'a2enmod headers >/dev/null && exec apache2-foreground')
docker "${args[@]}"
