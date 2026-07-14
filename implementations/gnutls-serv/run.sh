#!/usr/bin/env bash
set -euo pipefail

scenario="${PLAB_SCENARIO_ID:-tls.handshake.full}"
if [[ "$scenario" != "tls.handshake.full" ]]; then
  printf '{"schemaVersion":"protocol-lab.unsupported.v1","status":"unsupported","scenarioId":"%s","implementationId":"gnutls-serv","supportedScenarios":["tls.handshake.full"]}\n' "$scenario"
  exit 3
fi

image="${PLAB_GNUTLS_SERV_IMAGE:-incursa-protocol-lab-gnutls-serv:0.1.1}"
port="${PLAB_TARGET_PORT:-18462}"
skip_build="${PLAB_SKIP_BUILD:-false}"
proof_only="${PLAB_PROOF_ONLY:-false}"
root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [[ "${1:-}" == "--plan-only" ]]; then
  printf '{"schemaVersion":"protocol-lab.endpoint-plan.v1","implementationId":"gnutls-serv","packageVersion":"0.1.1","upstreamVersion":"3.8.9","scenarioId":"tls.handshake.full","image":"%s","hostPort":%s,"containerPort":8443,"controls":["TLS1.3","TLS_AES_128_GCM_SHA256","X25519","SIGN-ECDSA-SECP256R1-SHA256","protocol-lab-tls","fatal-sni","fatal-alpn","tickets-disabled"]}\n' "$image" "$port"
  exit 0
fi

cd "$root"
if [[ "$skip_build" != "true" ]]; then
  docker build --pull -f docker/Dockerfile -t "$image" .
fi
version_line="$(docker run --rm --entrypoint gnutls-serv "$image" --version | sed -n '1p')"
[[ "$version_line" == 'gnutls-serv 3.8.9' ]] || { printf 'Expected gnutls-serv 3.8.9, observed %s\n' "$version_line" >&2; exit 2; }
if [[ "$proof_only" == "true" ]]; then
  printf '{"status":"proved","image":"%s","upstreamVersion":"3.8.9"}\n' "$image"
  exit 0
fi
exec docker run --rm -p "$port:8443/tcp" "$image"
