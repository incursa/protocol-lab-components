#!/usr/bin/env bash
set -euo pipefail

scenario="${PLAB_SCENARIO_ID:-tls.handshake.full}"
if [[ "$scenario" != "tls.handshake.full" ]]; then
  printf '{"schemaVersion":"protocol-lab.unsupported.v1","status":"unsupported","scenarioId":"%s","implementationId":"wolfssl-tls13","supportedScenarios":["tls.handshake.full"]}\n' "$scenario"
  exit 3
fi

image="${PLAB_WOLFSSL_TLS13_IMAGE:-incursa-protocol-lab-wolfssl-tls13:0.1.0}"
port="${PLAB_TARGET_PORT:-18449}"
skip_build="${PLAB_SKIP_BUILD:-false}"
proof_only="${PLAB_PROOF_ONLY:-false}"
root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [[ "${1:-}" == "--plan-only" ]]; then
  printf '{"schemaVersion":"protocol-lab.endpoint-plan.v1","implementationId":"wolfssl-tls13","packageVersion":"0.1.0","upstreamVersion":"5.9.2","scenarioId":"tls.handshake.full","image":"%s","hostPort":%s,"containerPort":8443,"controls":["TLS1.3","TLS13-AES128-GCM-SHA256","X25519","ecdsa_secp256r1_sha256","protocol-lab-tls","tls.plab.test","tickets-disabled","zero-application-data"]}\n' "$image" "$port"
  exit 0
fi

cd "$root"
if [[ "$skip_build" != "true" ]]; then
  docker build --pull -f docker/Dockerfile -t "$image" .
fi
version_line="$(docker run --rm "$image" --version | sed -n '1p')"
[[ "$version_line" == 'wolfSSL 5.9.2' ]] || { printf 'Expected wolfSSL 5.9.2, observed %s\n' "$version_line" >&2; exit 2; }
if [[ "$proof_only" == "true" ]]; then
  printf '{"status":"proved","image":"%s","upstreamVersion":"5.9.2"}\n' "$image"
  exit 0
fi
exec docker run --rm -p "$port:8443/tcp" "$image"
