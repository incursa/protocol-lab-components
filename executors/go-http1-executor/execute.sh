#!/usr/bin/env bash
set -euo pipefail

target_base_url="${1:-${PLAB_TARGET_BASE_URL:-}}"
output_dir="${PLAB_ARTIFACT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/artifacts}"

if [[ -z "${target_base_url}" ]]; then
  echo "Target URL argument or PLAB_TARGET_BASE_URL is required." >&2
  exit 2
fi

mkdir -p "${output_dir}"
cd "$(dirname "${BASH_SOURCE[0]}")/source"
go run . --target-url "${target_base_url}" --output-dir "${output_dir}"
