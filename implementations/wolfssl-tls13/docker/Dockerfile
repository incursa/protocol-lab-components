FROM debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818 AS build

ARG WOLFSSL_VERSION=5.9.2
ARG WOLFSSL_ARCHIVE_SHA256=2f4ef3d4fd387a9b3191d36a6316d69116c46ff69bb9583b6c82b36d7b8ca114

RUN apt-get update \
    && apt-get install --yes --no-install-recommends autoconf automake build-essential ca-certificates curl libtool pkg-config \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /src
RUN curl --fail --location --retry 3 \
      --output wolfssl.tar.gz \
      "https://github.com/wolfSSL/wolfssl/archive/refs/tags/v${WOLFSSL_VERSION}-stable.tar.gz" \
    && echo "${WOLFSSL_ARCHIVE_SHA256}  wolfssl.tar.gz" | sha256sum --check - \
    && tar --extract --gzip --file wolfssl.tar.gz --strip-components=1 \
    && ./autogen.sh \
    && ./configure \
         --enable-tls13 --enable-alpn --enable-sni --enable-supportedcurves --enable-curve25519 \
         --disable-shared --enable-static CFLAGS='-O2' \
    && make --jobs=2 examples/server/server \
    && test "$(sh ./wolfssl-config --version)" = "${WOLFSSL_VERSION}" \
    && strip examples/server/server

FROM debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818
LABEL org.opencontainers.image.source="https://github.com/wolfSSL/wolfssl" \
      org.opencontainers.image.version="5.9.2" \
      org.opencontainers.image.revision="ac01707f552c611fbd135cc723b2682b3e7f80f2" \
      org.opencontainers.image.licenses="GPL-3.0-or-later"
COPY --from=build /src/examples/server/server /usr/local/bin/wolfssl-server
COPY --from=build /src/COPYING /usr/share/licenses/wolfssl/COPYING
COPY --from=build /src/certs/dh2048.pem /opt/wolfssl/certs/dh2048.pem
COPY docker/entrypoint.sh /usr/local/bin/protocol-lab-wolfssl-entrypoint
COPY certs/leaf.pem /opt/protocol-lab/certs/leaf.pem
COPY certs/leaf-key.pem /opt/protocol-lab/certs/leaf-key.pem
RUN chmod 0555 /usr/local/bin/wolfssl-server /usr/local/bin/protocol-lab-wolfssl-entrypoint \
    && chmod 0444 /opt/wolfssl/certs/dh2048.pem /opt/protocol-lab/certs/leaf.pem /opt/protocol-lab/certs/leaf-key.pem
ENV PLAB_TLS_PORT=8443 \
    PLAB_SCENARIO_ID=tls.handshake.full
EXPOSE 8443/tcp
WORKDIR /opt/wolfssl
USER 65532:65532
ENTRYPOINT ["protocol-lab-wolfssl-entrypoint"]
