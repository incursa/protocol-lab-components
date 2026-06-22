#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

for path in \
  "protocol-lab-package.json" \
  "protocol-lab.internal.json" \
  "scenarios/http3/external/peer-characterization.yaml" \
  "suites/http3-peer-characterization.yaml"; do
  test -f "$root/$path"
done

echo "HTTP/3 peer characterization scenario package files are present."
