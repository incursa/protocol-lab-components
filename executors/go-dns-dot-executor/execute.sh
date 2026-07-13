#!/usr/bin/env bash
set -euo pipefail
target="${PLAB_TARGET_BASE_URL:?PLAB_TARGET_BASE_URL is required}"
output="${PLAB_ARTIFACT_DIR:-artifacts}"
args=(--target-address "$target" --output-dir "$output")
if [[ -z "${PLAB_DURATION_SECONDS:-}" ]]; then args+=(--validation-only); fi
cd "$(dirname "$0")/source"
exec go run . "${args[@]}"
