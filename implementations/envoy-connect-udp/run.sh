#!/usr/bin/env bash
set -euo pipefail
scenario="${PLAB_SCENARIO_ID:-masque.connect-udp-tunnel}"
if [[ "$scenario" != "masque.connect-udp-tunnel" ]]; then
  printf '{"schemaVersion":"protocol-lab.unsupported.v1","status":"unsupported","scenarioId":"%s","implementationId":"envoy-connect-udp","supportedScenarios":["masque.connect-udp-tunnel"]}\n' "$scenario"
  exit 3
fi
image="${PLAB_ENVOY_CONNECT_UDP_IMAGE:-incursa-protocol-lab-envoy-connect-udp:0.1.1}"
port="${PLAB_TARGET_PORT:-5472}"
root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ "${1:-}" == "--plan-only" ]]; then
  printf '{"schemaVersion":"protocol-lab.endpoint-plan.v1","implementationId":"envoy-connect-udp","packageVersion":"0.1.1","upstreamVersion":"v1.38.3","scenarioId":"%s","image":"%s","hostPort":%s,"containerPort":4443,"protocol":"masque-connect-udp-over-h3","roles":["proxy","udp-target"],"upstreamStatus":"alpha"}\n' "$scenario" "$image" "$port"
  exit 0
fi
cd "$root"
if [[ "${PLAB_SKIP_BUILD:-false}" != "true" ]]; then docker build --pull -f docker/Dockerfile -t "$image" .; fi
[[ "$(docker run --rm "$image" --version)" == "envoy-connect-udp 0.1.1 envoy v1.38.3" ]]
exec docker run --rm -p "$port:4443/udp" "$image"
