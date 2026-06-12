#!/usr/bin/env bash
set -euo pipefail

port="${PLAB_HTTP_PORT:-${PORT:-8080}}"
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
run_root="${TMPDIR:-/tmp}/protocol-lab-nginx-http1-$$"
mkdir -p "$run_root"

command -v nginx >/dev/null 2>&1 || {
  echo "nginx executable was not found on PATH." >&2
  exit 1
}

sed "s/\${PLAB_HTTP_PORT}/$port/g" "$script_dir/nginx.conf.template" > "$run_root/nginx.conf"
nginx -p "$run_root" -c "$run_root/nginx.conf" -g 'daemon off;'
