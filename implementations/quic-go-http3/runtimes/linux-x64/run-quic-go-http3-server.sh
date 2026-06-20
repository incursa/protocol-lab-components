#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
server="$script_dir/quic-go-http3-server"

chmod +x "$server" 2>/dev/null || true
exec "$server" "$@"
