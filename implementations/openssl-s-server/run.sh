#!/usr/bin/env sh
set -eu

scenario="${PLAB_SCENARIO_ID:-tls.handshake.full}"
if [ "$scenario" != "tls.handshake.full" ]; then
  printf '{"schemaVersion":"protocol-lab.unsupported.v1","status":"unsupported","scenarioId":"%s","implementationId":"openssl-s-server","supportedScenarios":["tls.handshake.full"]}\n' "$scenario"
  exit 3
fi

listen_address="${PLAB_LISTEN_ADDRESS:-127.0.0.1:${PLAB_TARGET_PORT:-18461}}"
certificate="${PLAB_TLS_CERT_FILE:-$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)/certs/leaf.pem}"
private_key="${PLAB_TLS_KEY_FILE:-$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)/certs/leaf-key.pem}"
tool="${PLAB_OPENSSL_PATH:-openssl}"

if [ "${1:-}" = "--plan-only" ]; then
  printf '{"schemaVersion":"protocol-lab.endpoint-plan.v1","implementationId":"openssl-s-server","upstreamVersion":"3.3.0","scenarioId":"tls.handshake.full","listenAddress":"%s","executable":"%s","controls":["tls1.3","TLS_AES_128_GCM_SHA256","X25519","ecdsa_secp256r1_sha256","protocol-lab-tls","tickets-disabled"]}\n' "$listen_address" "$tool"
  exit 0
fi

version_line="$($tool version 2>&1 | sed -n '1p')"
case "$version_line" in
  "OpenSSL 3.3.0"|"OpenSSL 3.3.0 "*) ;;
  *) printf 'openssl-s-server requires OpenSSL 3.3.0; observed %s\n' "$version_line" >&2; exit 2 ;;
esac

exec "$tool" s_server -4 -accept "$listen_address" -cert "$certificate" -key "$private_key" -tls1_3 -ciphersuites TLS_AES_128_GCM_SHA256 -groups X25519 -sigalgs ecdsa_secp256r1_sha256 -alpn protocol-lab-tls -no_cache -no_ticket -num_tickets 0 -quiet -ign_eof
