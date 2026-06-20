#!/bin/sh
set -eu

version_output="$(nginx -V 2>&1)"
printf '%s\n' "$version_output"
case "$version_output" in
  *--with-http_v3_module*) ;;
  *)
    printf '%s\n' "ProtocolLab nginx HTTP/3 capability proof failed: nginx -V did not include --with_http_v3_module or --with-http_v3_module." >&2
    exit 78
    ;;
esac

mkdir -p /etc/nginx/certs
if [ ! -f /etc/nginx/certs/localhost.crt ] || [ ! -f /etc/nginx/certs/localhost.key ]; then
  openssl req \
    -x509 \
    -newkey rsa:2048 \
    -nodes \
    -days 2 \
    -subj "/CN=localhost" \
    -addext "subjectAltName=DNS:localhost,IP:127.0.0.1" \
    -keyout /etc/nginx/certs/localhost.key \
    -out /etc/nginx/certs/localhost.crt >/dev/null 2>&1
fi

plab_http_port="${PLAB_HTTP_PORT:-8443}"
case "$plab_http_port" in
  *[!0-9]*|'')
    printf '%s\n' "ProtocolLab nginx HTTP/3 configuration failed: PLAB_HTTP_PORT must be a numeric port." >&2
    exit 78
    ;;
esac
sed -i "s/__PLAB_HTTP_PORT__/${plab_http_port}/g" /etc/nginx/nginx.conf

nginx -t
exec "$@"
