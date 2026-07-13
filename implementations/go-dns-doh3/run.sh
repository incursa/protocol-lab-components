#!/usr/bin/env bash
set -euo pipefail
exec "$(dirname "$0")/bin/linux-x64/go-dns-doh3" "$@"
