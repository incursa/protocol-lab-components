#!/usr/bin/env bash
set -euo pipefail

package_root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
required_files=(
  "protocol-lab-package.json"
  "protocol-lab.internal.json"
  "scenarios/quic/transport/stream-throughput.yaml"
  "scenarios/quic/transport/latency-echo-1kb.yaml"
  "scenarios/quic/transport/multiplex-100-streams.yaml"
  "scenarios/quic/transport/stream-limits-100-streams.yaml"
  "scenarios/quic/transport/payload-large-1mb.yaml"
  "scenarios/quic/transport/duplex-streams.yaml"
  "scenarios/quic/transport/cancellation-reset-stream.yaml"
  "scenarios/quic/transport/cold-handshake.yaml"
  "scenarios/quic/transport/stream-churn.yaml"
  "scenarios/quic/transport/resumption-rejected.yaml"
  "scenarios/quic/transport/resumed-handshake.yaml"
  "scenarios/quic/transport/zero-rtt-accepted.yaml"
  "scenarios/quic/transport/zero-rtt-rejected.yaml"
  "suites/raw-quic-transport-v1-smoke.yaml"
)

for relative_path in "${required_files[@]}"; do
  if [[ ! -f "$package_root/$relative_path" ]]; then
    echo "Required raw QUIC scenario package file is missing: $relative_path" >&2
    exit 1
  fi
done

echo "Raw QUIC scenario package files are present."
