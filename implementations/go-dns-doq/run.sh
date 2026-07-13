#!/usr/bin/env sh
set -eu
exec "$(dirname "$0")/bin/linux-x64/go-dns-doq" --listen "${1:-127.0.0.1:18532}"
