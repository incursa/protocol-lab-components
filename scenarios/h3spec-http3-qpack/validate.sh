#!/usr/bin/env bash
set -euo pipefail

package_root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
required_files=(
  "protocol-lab-package.json"
  "protocol-lab.internal.json"
  "scenarios/http3/core/status.yaml"
  "scenarios/http3/headers/response-headers-50x32.yaml"
  "scenarios/http3/protocol/qpack-repeated-headers.yaml"
  "suites/h3spec-http3-qpack-focused.yaml"
)

for relative_path in "${required_files[@]}"; do
  if [[ ! -f "$package_root/$relative_path" ]]; then
    echo "Required h3spec scenario package file is missing: $relative_path" >&2
    exit 1
  fi
done

echo "h3spec HTTP/3 and QPACK scenario package files are present."
