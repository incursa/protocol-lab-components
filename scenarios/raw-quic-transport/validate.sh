#!/usr/bin/env bash
set -euo pipefail

package_root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
required_files=(
  "protocol-lab-package.json"
  "protocol-lab.internal.json"
  "scenarios/quic/transport/stream-throughput.yaml"
  "scenarios/quic/transport/stream-download-1mb.yaml"
  "scenarios/quic/transport/stream-throughput-64kb.yaml"
  "scenarios/quic/transport/stream-throughput-16mb.yaml"
  "scenarios/quic/transport/sustained-stream-256x64kb.yaml"
  "scenarios/quic/transport/sustained-stream-16384x1kb.yaml"
  "scenarios/quic/transport/sustained-download-256x64kb.yaml"
  "scenarios/quic/transport/sustained-download-16384x1kb.yaml"
  "scenarios/quic/transport/sustained-download-4096x1kb.yaml"
  "scenarios/quic/transport/latency-echo-1kb.yaml"
  "scenarios/quic/transport/multiplex-100-streams.yaml"
  "scenarios/quic/transport/multiplex-100x1kb.yaml"
  "scenarios/quic/transport/multiplex-16x1mb.yaml"
  "scenarios/quic/transport/multiplex-mixed-4x16.yaml"
  "scenarios/quic/transport/stream-limits-100-streams.yaml"
  "scenarios/quic/transport/flow-control-slow-reader-16x64kb.yaml"
  "scenarios/quic/transport/payload-large-1mb.yaml"
  "scenarios/quic/transport/duplex-streams.yaml"
  "scenarios/quic/transport/duplex-streams-16x1mb.yaml"
  "scenarios/quic/transport/duplex-streams-peer-matrix.yaml"
  "scenarios/quic/transport/cancellation-reset-stream.yaml"
  "scenarios/quic/transport/handshake-cold.yaml"
  "scenarios/quic/transport/connection-churn.yaml"
  "scenarios/quic/transport/stream-churn.yaml"
  "scenarios/quic/transport/resumption-rejected.yaml"
  "scenarios/quic/transport/resumed-handshake.yaml"
  "scenarios/quic/transport/zero-rtt-accepted.yaml"
  "scenarios/quic/transport/zero-rtt-rejected.yaml"
  "suites/raw-quic-transport-v1-smoke.yaml"
  "suites/quic-transport-v1-comparison.yaml"
)

for relative_path in "${required_files[@]}"; do
  if [[ ! -f "$package_root/$relative_path" ]]; then
    echo "Required raw QUIC scenario package file is missing: $relative_path" >&2
    exit 1
  fi
done

echo "Raw QUIC scenario package files are present."
