FROM debian:12.12-slim@sha256:d5d3f9c23164ea16f31852f95bd5959aad1c5e854332fe00f7b3a20fcc9f635c AS builder

ARG H2O_COMMIT=edd7a120bfc4af11ac0cbebce2a43cc1f93f9af1

RUN apt-get update \
    && apt-get install --no-install-recommends -y \
        build-essential \
        ca-certificates \
        cmake \
        git \
        libssl-dev \
        pkg-config \
        zlib1g-dev \
    && rm -rf /var/lib/apt/lists/*

RUN git clone --depth 1 --recurse-submodules https://github.com/h2o/h2o.git /src/h2o \
    && test "$(git -C /src/h2o rev-parse HEAD)" = "$H2O_COMMIT" \
    && cmake -S /src/h2o -B /src/h2o/build \
        -DCMAKE_BUILD_TYPE=Release \
        -DWITH_DTRACE=OFF \
        -DWITH_IO_URING=OFF \
        -DWITH_KTLS=OFF \
        -DWITH_MRUBY=OFF \
    && cmake --build /src/h2o/build --parallel \
    && cmake --install /src/h2o/build --prefix /opt/h2o

FROM debian:12.12-slim@sha256:d5d3f9c23164ea16f31852f95bd5959aad1c5e854332fe00f7b3a20fcc9f635c

RUN apt-get update \
    && apt-get install --no-install-recommends -y \
        ca-certificates \
        libssl3 \
        openssl \
        zlib1g \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p /etc/h2o/certs /srv/h2o/bytes

COPY --from=builder /opt/h2o/bin/h2o /usr/local/bin/h2o
COPY --from=builder /opt/h2o/share/h2o /usr/local/share/h2o
COPY h2o.conf /etc/h2o/h2o.conf
COPY docker-entrypoint.sh /usr/local/bin/protocol-lab-h2o-http3-entrypoint.sh

RUN head -c 1024 /dev/zero | tr '\0' 'x' > /srv/h2o/bytes/1024.bin \
    && head -c 65536 /dev/zero | tr '\0' 'x' > /srv/h2o/bytes/65536.bin \
    && printf '%s\n' '{"protocol":"h3","server":"h2o","implementation":"h2o-http3","utc":"2026-07-17T00:00:00Z","processId":1}' > /srv/h2o/status.json \
    && chmod +x /usr/local/bin/protocol-lab-h2o-http3-entrypoint.sh

EXPOSE 8443/tcp
EXPOSE 8443/udp

ENTRYPOINT ["/usr/local/bin/protocol-lab-h2o-http3-entrypoint.sh"]
CMD ["h2o", "-c", "/etc/h2o/h2o.conf"]
