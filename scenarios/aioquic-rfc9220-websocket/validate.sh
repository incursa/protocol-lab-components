#!/usr/bin/env bash
set -euo pipefail

package_root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
required_files=(
  "protocol-lab-package.json"
  "protocol-lab.internal.json"
  "scenarios/http3/websocket/rfc9220-extended-connect.yaml"
  "scenarios/http3/websocket/rfc9220-control-frames.yaml"
  "scenarios/http3/websocket/rfc9220-text-echo.yaml"
  "scenarios/http3/websocket/rfc9220-binary-echo.yaml"
  "scenarios/http3/websocket/rfc9220-close.yaml"
  "suites/aioquic-rfc9220-websocket-proof.yaml"
)

for relative_path in "${required_files[@]}"; do
  if [[ ! -f "$package_root/$relative_path" ]]; then
    echo "Required aioquic RFC9220 scenario package file is missing: $relative_path" >&2
    exit 1
  fi
done

echo "aioquic RFC9220 WebSocket scenario package files are present."
