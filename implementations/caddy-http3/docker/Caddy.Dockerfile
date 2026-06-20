FROM caddy:2.11.2-alpine

COPY Caddyfile /etc/caddy/Caddyfile
RUN mkdir -p /srv/bytes \
    && head -c 1024 /dev/zero | tr '\0' 'x' > /srv/bytes/1024 \
    && head -c 65536 /dev/zero | tr '\0' 'x' > /srv/bytes/65536

EXPOSE 8443/tcp
EXPOSE 8443/udp
