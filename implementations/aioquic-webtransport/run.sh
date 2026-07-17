#!/usr/bin/env bash
set -euo pipefail
scenario="${PLAB_SCENARIO_ID:-webtransport.session-bidi-echo}"
if [[ "$scenario" != "webtransport.session-bidi-echo" ]]; then
  printf '{"schemaVersion":"protocol-lab.unsupported.v1","status":"unsupported","scenarioId":"%s","implementationId":"aioquic-webtransport","supportedScenarios":["webtransport.session-bidi-echo"]}\n' "$scenario"
  exit 3
fi
image="${PLAB_AIOQUIC_WEBTRANSPORT_IMAGE:-incursa-protocol-lab-aioquic-webtransport:0.1.1}"
port="${PLAB_TARGET_PORT:-5462}"
root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ "${1:-}" == "--plan-only" ]]; then
  printf '{"schemaVersion":"protocol-lab.endpoint-plan.v1","implementationId":"aioquic-webtransport","packageVersion":"0.1.1","upstreamVersion":"1.3.0","scenarioId":"%s","image":"%s","hostPort":%s,"containerPort":4433,"protocol":"webtransport-over-h3"}\n' "$scenario" "$image" "$port"
  exit 0
fi
cd "$root"
if [[ "${PLAB_SKIP_BUILD:-false}" != "true" ]]; then docker build --pull -f docker/Dockerfile -t "$image" .; fi
[[ "$(docker run --rm "$image" --version)" == "aioquic-webtransport 0.1.1 aioquic 1.3.0" ]]
exec docker run --rm -p "$port:4433/udp" "$image"
