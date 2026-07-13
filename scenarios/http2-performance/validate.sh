#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${root}"
printf '%s  %s\n' 'c99e4cab988b9d4921122a45fa5a0da699d2711e6b689528effb899323a9682c' 'scenarios/http2/core/plaintext.yaml' | sha256sum --check --status
printf '%s  %s\n' '082a24ef29914cb9cffd1d4b1f66ae512bf3d49d3a683963e434a2a6a29ba7db' 'scenarios/http2/core/json.yaml' | sha256sum --check --status
printf '%s  %s\n' '453c29b24d2030b6317ae4f7256e915db6056ac040871b414b0cb3beb9ae279d' 'load-profiles/http2-smoke.yaml' | sha256sum --check --status
echo 'Validated HTTP/2 scenario package authority lock at a4dcd74e5c8907907ccc58808da92d2b920b2fbc.'
