FROM debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818 AS build

ARG OPENSSL_VERSION=3.3.0
ARG OPENSSL_ARCHIVE_SHA256=53e66b043322a606abf0087e7699a0e033a37fa13feb9742df35c3a33b18fb02

RUN apt-get update \
    && apt-get install --yes --no-install-recommends build-essential ca-certificates curl perl \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /src
RUN curl --fail --location --retry 3 \
      --output openssl.tar.gz \
      "https://github.com/openssl/openssl/releases/download/openssl-${OPENSSL_VERSION}/openssl-${OPENSSL_VERSION}.tar.gz" \
    && echo "${OPENSSL_ARCHIVE_SHA256}  openssl.tar.gz" | sha256sum --check - \
    && tar --extract --gzip --file openssl.tar.gz --strip-components=1 \
    && ./Configure linux-x86_64 no-shared no-tests --prefix=/opt/openssl --openssldir=/opt/openssl/ssl \
    && make -j2 \
    && make install_sw \
    && /opt/openssl/bin/openssl version | grep -E '^OpenSSL 3\.3\.0([[:space:]]|$)'

FROM debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818
LABEL org.opencontainers.image.source="https://github.com/openssl/openssl" \
      org.opencontainers.image.version="3.3.0" \
      org.opencontainers.image.revision="4cb31128b5790819dfeea2739fbde265f71a10a2" \
      org.opencontainers.image.licenses="Apache-2.0"
COPY --from=build /opt/openssl /opt/openssl
COPY --from=build /src/LICENSE.txt /usr/share/licenses/openssl/LICENSE.txt
COPY certs/leaf.pem /opt/protocol-lab/certs/leaf.pem
COPY certs/leaf-key.pem /opt/protocol-lab/certs/leaf-key.pem
RUN chmod 0444 /opt/protocol-lab/certs/leaf.pem /opt/protocol-lab/certs/leaf-key.pem
ENV PATH="/opt/openssl/bin:${PATH}"
EXPOSE 8443/tcp
USER 65532:65532
ENTRYPOINT ["openssl", "s_server"]
CMD ["-4", "-accept", "0.0.0.0:8443", "-cert", "/opt/protocol-lab/certs/leaf.pem", "-key", "/opt/protocol-lab/certs/leaf-key.pem", "-tls1_3", "-ciphersuites", "TLS_AES_128_GCM_SHA256", "-groups", "X25519", "-sigalgs", "ecdsa_secp256r1_sha256", "-alpn", "protocol-lab-tls", "-no_cache", "-no_ticket", "-num_tickets", "0", "-quiet", "-ign_eof"]
