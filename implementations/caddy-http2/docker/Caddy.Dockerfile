FROM caddy:2.11.2-alpine@sha256:834468128c7696cec0ceea6172f7d692daf645ae51983ca76e39da54a97c570d

LABEL org.opencontainers.image.source="https://github.com/caddyserver/caddy" \
      org.opencontainers.image.version="2.11.2" \
      org.opencontainers.image.licenses="Apache-2.0"

COPY Caddyfile /etc/caddy/Caddyfile
COPY third-party/caddy-LICENSE.txt /usr/share/licenses/caddy/LICENSE

EXPOSE 8080/tcp

CMD ["caddy", "run", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile"]
