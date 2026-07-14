FROM ubuntu/apache2@sha256:6563a8f98ce5469715962cf217335ec73842e56abb3720094a15f2b6747b87bc

RUN a2enmod headers http2 ssl >/dev/null
COPY apache-http2-h2c.conf /etc/apache2/conf-enabled/zzz-protocol-lab-http2-h2c.conf
COPY apache-http2-tls.conf /etc/apache2/conf-enabled/zzz-protocol-lab-http2-tls.conf
COPY certs /run/protocol-lab-certs
COPY fixtures /tmp/protocol-lab-fixtures
RUN set -eux; \
    mkdir -p /var/www/protocol-lab; \
    for encoded in /tmp/protocol-lab-fixtures/*.b64; do \
      base64 --decode "$encoded" > "/var/www/protocol-lab/$(basename "${encoded%.b64}")"; \
    done; \
    rm -rf /tmp/protocol-lab-fixtures

EXPOSE 8082 8443
