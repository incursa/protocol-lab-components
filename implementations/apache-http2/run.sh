#!/usr/bin/env bash
set -euo pipefail

image='ubuntu/apache2@sha256:6563a8f98ce5469715962cf217335ec73842e56abb3720094a15f2b6747b87bc'
variant="${PLAB_PROTOCOL_VARIANT:-h2c}"
container_name="${PLAB_CONTAINER_NAME:-protocol-lab-apache-http2-$$}"
case "$variant" in
  h2c|http2-h2c-prior-knowledge)
    normalized_variant='h2c'; port="${PLAB_HTTP_PORT:-${PORT:-8082}}"; container_port='8082'; config_name='apache-http2-h2c.conf'; module_command='a2enmod headers http2 >/dev/null && exec apache2-foreground' ;;
  tls-alpn|http2-tls-alpn)
    normalized_variant='tls-alpn'; port="${PLAB_HTTPS_PORT:-${PORT:-8443}}"; container_port='8443'; config_name='apache-http2-tls.conf'; module_command='a2enmod headers http2 ssl >/dev/null && exec apache2-foreground' ;;
  *) echo "Unsupported Apache HTTP/2 execution variant '$variant'." >&2; exit 2 ;;
esac

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
run_root="$(mktemp -d "${TMPDIR:-/tmp}/protocol-lab-apache-http2.XXXXXX")"
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
  --publish "127.0.0.1:${port}:${container_port}"
  --mount "type=bind,source=$fixture_root,target=/var/www/protocol-lab,readonly"
  --mount "type=bind,source=$script_dir/$config_name,target=/etc/apache2/conf-enabled/zzz-protocol-lab-http2.conf,readonly")
if [[ "$normalized_variant" == 'tls-alpn' ]]; then args+=(--mount "type=bind,source=$script_dir/certs,target=/run/protocol-lab-certs,readonly"); fi
args+=(--name "$container_name")
args+=(--entrypoint /bin/sh "$image" -ec "$module_command")
docker "${args[@]}"
