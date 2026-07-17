#!/usr/bin/env bash
set -euo pipefail
scenario="${PLAB_SCENARIO_ID:-masque.connect-udp-tunnel}"
if [[ "$scenario" != "masque.connect-udp-tunnel" ]]; then
  printf '{"schemaVersion":"protocol-lab.unsupported.v1","status":"unsupported","scenarioId":"%s","implementationId":"masque-go-connect-udp","supportedScenarios":["masque.connect-udp-tunnel"]}\n' "$scenario"
  exit 3
fi
image="${PLAB_MASQUE_GO_IMAGE:-incursa-protocol-lab-masque-go-connect-udp:0.1.1}"
port="${PLAB_TARGET_PORT:-5471}"
root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ "${1:-}" == "--plan-only" ]]; then
  printf '{"schemaVersion":"protocol-lab.endpoint-plan.v1","implementationId":"masque-go-connect-udp","packageVersion":"0.1.1","upstreamVersion":"v0.4.0","scenarioId":"%s","image":"%s","hostPort":%s,"containerPort":4443,"protocol":"masque-connect-udp-over-h3","roles":["proxy","udp-target"]}\n' "$scenario" "$image" "$port"
  exit 0
fi
cd "$root"
if [[ "${PLAB_SKIP_BUILD:-false}" != "true" ]]; then docker build --pull -f docker/Dockerfile -t "$image" .; fi
[[ "$(docker run --rm "$image" --version)" == "masque-go-connect-udp 0.1.1 masque-go v0.4.0" ]]
exec docker run --rm -e "PLAB_PUBLIC_PORT=$port" -p "$port:4443/udp" "$image"
