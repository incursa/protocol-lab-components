#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export PLAB_LISTEN_ADDRESS="127.0.0.1:${PLAB_TARGET_PORT:-18444}"
cd "$root/source"
exec go run .
