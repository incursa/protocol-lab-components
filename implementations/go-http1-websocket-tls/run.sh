#!/usr/bin/env bash
set -euo pipefail
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export PLAB_TLS_CERTIFICATE_PATH="$script_dir/certs/leaf.pem"
export PLAB_TLS_PRIVATE_KEY_PATH="$script_dir/certs/leaf-key.pem"
exec "$script_dir/bin/linux-x64/go-http1-websocket-tls" "$@"
