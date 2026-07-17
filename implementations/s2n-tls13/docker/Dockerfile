FROM debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818 AS build

ARG S2N_TLS_VERSION=1.7.5
ARG S2N_TLS_ARCHIVE_SHA256=841f67cb1eb4ce71c829fa12ce940ae33556429c7e137be5c16461999bb7f666

RUN apt-get update \
    && apt-get install --yes --no-install-recommends build-essential ca-certificates cmake curl libssl-dev ninja-build \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /src
RUN curl --fail --location --retry 3 \
      --output s2n-tls.tar.gz \
      "https://github.com/aws/s2n-tls/archive/refs/tags/v${S2N_TLS_VERSION}.tar.gz" \
    && echo "${S2N_TLS_ARCHIVE_SHA256}  s2n-tls.tar.gz" | sha256sum --check - \
    && tar --extract --gzip --file s2n-tls.tar.gz --strip-components=1 \
    && cmake -S . -B build -G Ninja \
         -DBUILD_SHARED_LIBS=OFF -DBUILD_TESTING=OFF -DS2N_INTERN_LIBCRYPTO=OFF \
         -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=/opt/s2n \
    && cmake --build build --parallel 2 \
    && cmake --install build
COPY source/main.c /adapter/main.c
RUN cc -std=c11 -O2 -Wall -Wextra -Wpedantic -Werror \
      -I/opt/s2n/include /adapter/main.c /opt/s2n/lib/libs2n.a \
      -lcrypto -ldl -lpthread -o /opt/protocol-lab-s2n-tls13 \
    && /opt/protocol-lab-s2n-tls13 --version | grep -Fx 's2n-tls 1.7.5'

FROM debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818
LABEL org.opencontainers.image.source="https://github.com/aws/s2n-tls" \
      org.opencontainers.image.version="1.7.5" \
      org.opencontainers.image.revision="f5f6c6c2ce2370de1aa3ade6899a7321d1127bb8" \
      org.opencontainers.image.licenses="Apache-2.0"
RUN apt-get update \
    && apt-get install --yes --no-install-recommends libssl3 \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /opt/protocol-lab-s2n-tls13 /usr/local/bin/protocol-lab-s2n-tls13
COPY --from=build /src/LICENSE /usr/share/licenses/s2n-tls/LICENSE
COPY certs/leaf.pem /opt/protocol-lab/certs/leaf.pem
COPY certs/leaf-key.pem /opt/protocol-lab/certs/leaf-key.pem
RUN chmod 0555 /usr/local/bin/protocol-lab-s2n-tls13 \
    && chmod 0444 /opt/protocol-lab/certs/leaf.pem /opt/protocol-lab/certs/leaf-key.pem
ENV PLAB_TLS_PORT=8443 \
    PLAB_SCENARIO_ID=tls.handshake.full
EXPOSE 8443/tcp
USER 65532:65532
ENTRYPOINT ["protocol-lab-s2n-tls13"]
