#!/usr/bin/env bash
set -euo pipefail

port="${PLAB_HTTP_PORT:-${PORT:-8080}}"
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

export PLAB_HTTP_PORT="$port"
command -v caddy >/dev/null 2>&1 || {
  echo "caddy executable was not found on PATH." >&2
  exit 1
}

caddy run --config "$script_dir/Caddyfile" --adapter caddyfile
