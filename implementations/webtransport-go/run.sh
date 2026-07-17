#!/usr/bin/env bash
set -euo pipefail
scenario="${PLAB_SCENARIO_ID:-webtransport.session-bidi-echo}"
if [[ "$scenario" != "webtransport.session-bidi-echo" && "$scenario" != "webtransport.session-datagram-echo" ]]; then
  printf '{"schemaVersion":"protocol-lab.unsupported.v1","status":"unsupported","scenarioId":"%s","implementationId":"webtransport-go","supportedScenarios":["webtransport.session-bidi-echo","webtransport.session-datagram-echo"]}\n' "$scenario"
  exit 3
fi
image="${PLAB_WEBTRANSPORT_GO_IMAGE:-incursa-protocol-lab-webtransport-go:0.1.2}"
port="${PLAB_TARGET_PORT:-5461}"
root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ "${1:-}" == "--plan-only" ]]; then
  printf '{"schemaVersion":"protocol-lab.endpoint-plan.v1","implementationId":"webtransport-go","packageVersion":"0.1.2","upstreamVersion":"v0.11.1","scenarioId":"%s","image":"%s","hostPort":%s,"containerPort":4433,"protocol":"webtransport-over-h3"}\n' "$scenario" "$image" "$port"
  exit 0
fi
cd "$root"
if [[ "${PLAB_SKIP_BUILD:-false}" != "true" ]]; then docker build --pull -f docker/Dockerfile -t "$image" .; fi
[[ "$(docker run --rm "$image" --version)" == "webtransport-go 0.1.2 webtransport-go v0.11.1" ]]
exec docker run --rm -p "$port:4433/udp" "$image"
