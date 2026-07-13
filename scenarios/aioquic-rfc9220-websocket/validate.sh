#!/usr/bin/env bash
set -euo pipefail

package_root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
required_files=(
  "protocol-lab-package.json"
  "protocol-lab.internal.json"
  "authority-lock.json"
  "scenarios/http3/websocket/rfc9220-extended-connect.yaml"
  "scenarios/http3/websocket/rfc9220-control-frames.yaml"
  "scenarios/http3/websocket/rfc9220-text-echo.yaml"
  "scenarios/http3/websocket/rfc9220-binary-echo.yaml"
  "scenarios/http3/websocket/rfc9220-close.yaml"
  "scenarios/http3/websocket/rfc9220-fragmented-binary-echo.yaml"
  "suites/aioquic-rfc9220-websocket-proof.yaml"
  "tests/test_authority_parity.py"
)

for relative_path in "${required_files[@]}"; do
  if [[ ! -f "$package_root/$relative_path" ]]; then
    echo "Required aioquic RFC9220 scenario package file is missing: $relative_path" >&2
    exit 1
  fi
done

grep -q '8c4bbe8b7ee94b0e53427dd5ac15e7ede7b77574' "$package_root/authority-lock.json"
python3 "$package_root/tests/test_authority_parity.py" --scenario-root "$package_root"

echo "aioquic RFC9220 WebSocket scenario package authority lock is valid."
