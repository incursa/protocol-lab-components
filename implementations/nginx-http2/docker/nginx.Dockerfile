FROM nginx:1.29.0-alpine@sha256:d67ea0d64d518b1bb04acde3b00f722ac3e9764b3209a9b0a98924ba35e4b779

LABEL org.opencontainers.image.source="https://github.com/nginx/nginx" \
      org.opencontainers.image.version="1.29.0" \
      org.opencontainers.image.licenses="BSD-2-Clause"

COPY nginx.conf /etc/nginx/nginx.conf
COPY third-party/nginx-LICENSE.txt /usr/share/licenses/nginx/LICENSE

EXPOSE 8080/tcp

CMD ["nginx", "-g", "daemon off;"]
