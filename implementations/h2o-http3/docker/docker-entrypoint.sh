#!/usr/bin/env bash
set -euo pipefail

export H2O_ROOT=/usr/local

if [[ ! -s /etc/h2o/certs/localhost.crt || ! -s /etc/h2o/certs/localhost.key ]]; then
  openssl req -x509 -newkey rsa:2048 -nodes -days 7 \
    -subj /CN=localhost \
    -addext 'subjectAltName=DNS:localhost,DNS:host.docker.internal,DNS:plab-worker-sut-01,DNS:plab-worker-load-01' \
    -keyout /etc/h2o/certs/localhost.key \
    -out /etc/h2o/certs/localhost.crt \
    >/dev/null 2>&1
fi

exec "$@"
