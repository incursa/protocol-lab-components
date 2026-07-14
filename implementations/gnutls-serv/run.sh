#!/usr/bin/env sh
set -eu

scenario="${PLAB_SCENARIO_ID:-tls.handshake.full}"
if [ "$scenario" != "tls.handshake.full" ]; then
  printf '{"schemaVersion":"protocol-lab.unsupported.v1","status":"unsupported","scenarioId":"%s","implementationId":"gnutls-serv","supportedScenarios":["tls.handshake.full"]}\n' "$scenario"
  exit 3
fi

port="${PLAB_TARGET_PORT:-18462}"
if [ -n "${PLAB_LISTEN_ADDRESS:-}" ]; then
  case "$PLAB_LISTEN_ADDRESS" in
    *:*) port="${PLAB_LISTEN_ADDRESS##*:}" ;;
    *) printf 'PLAB_LISTEN_ADDRESS must be host:port; observed %s\n' "$PLAB_LISTEN_ADDRESS" >&2; exit 2 ;;
  esac
fi
case "$port" in *[!0-9]*|'') printf 'TLS port must be numeric; observed %s\n' "$port" >&2; exit 2 ;; esac

root="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
certificate="${PLAB_TLS_CERT_FILE:-$root/certs/leaf.pem}"
private_key="${PLAB_TLS_KEY_FILE:-$root/certs/leaf-key.pem}"
tool="${PLAB_GNUTLS_SERV_PATH:-gnutls-serv}"
priority='NORMAL:-VERS-ALL:+VERS-TLS1.3:-CIPHER-ALL:+AES-128-GCM:-GROUP-ALL:+GROUP-X25519:-SIGN-ALL:+SIGN-ECDSA-SECP256R1-SHA256'

if [ "${1:-}" = "--plan-only" ]; then
  printf '{"schemaVersion":"protocol-lab.endpoint-plan.v1","implementationId":"gnutls-serv","upstreamVersion":"3.8.9","scenarioId":"tls.handshake.full","port":%s,"executable":"%s","controls":["TLS1.3","TLS_AES_128_GCM_SHA256","X25519","SIGN-ECDSA-SECP256R1-SHA256","protocol-lab-tls","fatal-sni","fatal-alpn","tickets-disabled"]}\n' "$port" "$tool"
  exit 0
fi

version_line="$($tool --version 2>&1 | sed -n '1p')"
if [ "$version_line" != 'gnutls-serv 3.8.9' ]; then
  printf 'gnutls-serv wrapper requires gnutls-serv 3.8.9; observed %s\n' "$version_line" >&2
  exit 2
fi

exec "$tool" --port "$port" --x509certfile "$certificate" --x509keyfile "$private_key" --priority "$priority" --alpn protocol-lab-tls --alpn-fatal --sni-hostname tls.plab.test --sni-hostname-fatal --disable-client-cert --noticket --echo --quiet
