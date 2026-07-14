FROM debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818 AS build

ARG GNUTLS_VERSION=3.8.9
ARG GNUTLS_ARCHIVE_SHA256=69e113d802d1670c4d5ac1b99040b1f2d5c7c05daec5003813c049b5184820ed

RUN apt-get update \
    && apt-get install --yes --no-install-recommends \
      build-essential ca-certificates curl libgmp-dev libidn2-0-dev nettle-dev \
      libtasn1-6-dev libunistring-dev pkg-config xz-utils \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /src
RUN curl --fail --location --retry 3 \
      --output gnutls.tar.xz \
      "https://www.gnupg.org/ftp/gcrypt/gnutls/v3.8/gnutls-${GNUTLS_VERSION}.tar.xz" \
    && echo "${GNUTLS_ARCHIVE_SHA256}  gnutls.tar.xz" | sha256sum --check - \
    && tar --extract --xz --file gnutls.tar.xz --strip-components=1 \
    && ./configure --prefix=/opt/gnutls --disable-doc --disable-tests --disable-guile --disable-cxx --without-p11-kit \
    && make -j2 \
    && make install \
    && LD_LIBRARY_PATH=/opt/gnutls/lib /opt/gnutls/bin/gnutls-serv --version | head -n 1 | grep -Fx 'gnutls-serv 3.8.9'

FROM debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818
LABEL org.opencontainers.image.source="https://gitlab.com/gnutls/gnutls" \
      org.opencontainers.image.version="3.8.9" \
      org.opencontainers.image.revision="477a733247460b94cd2b37a10579c27ca6fc196f" \
      org.opencontainers.image.licenses="GPL-3.0-or-later"
RUN apt-get update \
    && apt-get install --yes --no-install-recommends libgmp10 libidn2-0 libnettle8 libhogweed6 libtasn1-6 libunistring2 \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /opt/gnutls /opt/gnutls
COPY --from=build /src/COPYING /usr/share/licenses/gnutls/COPYING
COPY certs/leaf.pem /opt/protocol-lab/certs/leaf.pem
COPY certs/leaf-key.pem /opt/protocol-lab/certs/leaf-key.pem
RUN chmod 0444 /opt/protocol-lab/certs/leaf.pem /opt/protocol-lab/certs/leaf-key.pem
ENV PATH="/opt/gnutls/bin:${PATH}" \
    LD_LIBRARY_PATH="/opt/gnutls/lib"
EXPOSE 8443/tcp
USER 65532:65532
ENTRYPOINT ["gnutls-serv"]
CMD ["--port", "8443", "--x509certfile", "/opt/protocol-lab/certs/leaf.pem", "--x509keyfile", "/opt/protocol-lab/certs/leaf-key.pem", "--priority", "NORMAL:-VERS-ALL:+VERS-TLS1.3:-CIPHER-ALL:+AES-128-GCM:-GROUP-ALL:+GROUP-X25519:-SIGN-ALL:+SIGN-ECDSA-SECP256R1-SHA256", "--alpn", "protocol-lab-tls", "--alpn-fatal", "--sni-hostname", "tls.plab.test", "--sni-hostname-fatal", "--disable-client-cert", "--noticket", "--echo", "--quiet"]
