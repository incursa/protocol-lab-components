FROM nginx:1.29.0-alpine@sha256:d67ea0d64d518b1bb04acde3b00f722ac3e9764b3209a9b0a98924ba35e4b779

RUN apk add --no-cache openssl=3.5.7-r0 \
    && mkdir -p /srv/bytes \
    && head -c 1024 /dev/zero | tr '\0' 'x' > /srv/bytes/1024 \
    && head -c 65536 /dev/zero | tr '\0' 'x' > /srv/bytes/65536

COPY nginx.conf /etc/nginx/nginx.conf
COPY docker-entrypoint.sh /usr/local/bin/protocol-lab-nginx-http3-entrypoint.sh

RUN chmod +x /usr/local/bin/protocol-lab-nginx-http3-entrypoint.sh

EXPOSE 8443/tcp
EXPOSE 8443/udp

ENTRYPOINT ["/usr/local/bin/protocol-lab-nginx-http3-entrypoint.sh"]
CMD ["nginx", "-g", "daemon off;"]
