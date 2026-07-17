#!/usr/bin/env sh
set -eu

if [ "${1:-}" = "--version" ]; then
  printf 'wolfSSL 5.9.2\n'
  exit 0
fi

printf '{"eventName":"ready","implementationId":"wolfssl-tls13","implementationVersion":"0.1.0","scenarioId":"tls.handshake.full","tlsVersion":"TLS1.3","cipherSuite":"TLS_AES_128_GCM_SHA256","keyExchangeGroup":"X25519","alpn":"protocol-lab-tls","serverName":"tls.plab.test","sessionTicketsEnabled":false}\n'
exec /usr/local/bin/wolfssl-server \
  -b -i -d -T -t \
  -v 4 \
  -l TLS13-AES128-GCM-SHA256 \
  -L F:protocol-lab-tls \
  -S tls.plab.test \
  -p "${PLAB_TLS_PORT:-8443}" \
  -c /opt/protocol-lab/certs/leaf.pem \
  -k /opt/protocol-lab/certs/leaf-key.pem
