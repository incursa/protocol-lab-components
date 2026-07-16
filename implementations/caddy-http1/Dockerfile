FROM caddy@sha256:834468128c7696cec0ceea6172f7d692daf645ae51983ca76e39da54a97c570d

COPY Caddyfile /etc/caddy/Caddyfile

ENV PLAB_HTTP_PORT=8080
EXPOSE 8080

CMD ["run", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile"]
