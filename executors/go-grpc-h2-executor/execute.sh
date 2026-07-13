#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
: "${PLAB_TARGET_BASE_URL:?PLAB_TARGET_BASE_URL is required}"
output="${PLAB_ARTIFACT_DIR:-$root/artifacts}"
mkdir -p "$output"
cd "$root/source"
exec go run . --target-url "$PLAB_TARGET_BASE_URL" --output-dir "$output"
