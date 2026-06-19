#!/usr/bin/env bash
set -euo pipefail

package_root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
binary="$package_root/bin/linux-x64/quic-go-raw-load"

if [[ -x "$binary" ]]; then
  exec "$binary" "$@"
fi

source_root="$package_root/source"
if [[ ! -f "$source_root/go.mod" ]]; then
  echo "quic-go raw load binary and source payload were not found." >&2
  exit 1
fi

exec go -C "$source_root" run ./cmd/quic-go-raw-load "$@"
