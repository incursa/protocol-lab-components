#!/usr/bin/env bash
set -euo pipefail
export PLAB_DOT_LISTEN="127.0.0.1:${PLAB_DOT_PORT:-18530}"
cd "$(dirname "$0")"
exec ./bin/linux-x64/go-dns-dot
