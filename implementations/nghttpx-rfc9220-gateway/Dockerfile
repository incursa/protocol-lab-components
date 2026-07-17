FROM debian@sha256:b6e2a152f22a40ff69d92cb397223c906017e1391a73c952b588e51af8883bf8 AS builder

ARG NGHTTP2_COMMIT=68cb6900fde14c77f0cd7add0e094a862960eb99
ARG NGTCP2_COMMIT=716e64b05f4a3709dfc0b0522cf9fd4456d055e5
ARG NGHTTP3_COMMIT=d326f4c1eb3f6a780d77793b30e16756c498f913
ARG AWS_LC_COMMIT=d50ded594c835fd277997461815ad473949f2c7a

RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        autoconf automake build-essential ca-certificates cmake git libc-ares-dev libev-dev \
        libjemalloc-dev libtool ninja-build pkg-config python3 zlib1g-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src
RUN git clone https://github.com/aws/aws-lc.git aws-lc \
    && cd aws-lc \
    && git checkout "$AWS_LC_COMMIT" \
    && cmake -S . -B build -GNinja -DCMAKE_BUILD_TYPE=Release -DDISABLE_GO=ON -DCMAKE_INSTALL_PREFIX=/opt/aws-lc \
    && cmake --build build --parallel \
    && cmake --install build

RUN git clone https://github.com/ngtcp2/nghttp3.git nghttp3 \
    && cd nghttp3 \
    && git checkout "$NGHTTP3_COMMIT" \
    && git submodule update --init --depth 1 \
    && autoreconf -i \
    && ./configure --prefix=/opt/nghttp3 --enable-lib-only \
    && make -j"$(nproc)" \
    && make install

RUN git clone https://github.com/ngtcp2/ngtcp2.git ngtcp2 \
    && cd ngtcp2 \
    && git checkout "$NGTCP2_COMMIT" \
    && git submodule update --init --depth 1 \
    && autoreconf -i \
    && PKG_CONFIG_PATH=/opt/aws-lc/lib/pkgconfig:/opt/nghttp3/lib/pkgconfig \
       ./configure --prefix=/opt/ngtcp2 --enable-lib-only --with-boringssl \
         BORINGSSL_CFLAGS=-I/opt/aws-lc/include \
         BORINGSSL_LIBS='-L/opt/aws-lc/lib -lssl -lcrypto' \
    && make -j"$(nproc)" \
    && make install

RUN git clone https://github.com/nghttp2/nghttp2.git nghttp2 \
    && cd nghttp2 \
    && git checkout "$NGHTTP2_COMMIT" \
    && git submodule update --init --depth 1 \
    && autoreconf -i \
    && PKG_CONFIG_PATH=/opt/aws-lc/lib/pkgconfig:/opt/nghttp3/lib/pkgconfig:/opt/ngtcp2/lib/pkgconfig \
       LDFLAGS='-Wl,-rpath,/opt/aws-lc/lib -Wl,-rpath,/opt/nghttp3/lib -Wl,-rpath,/opt/ngtcp2/lib' \
       ./configure --prefix=/opt/nghttp2 --enable-http3 --disable-python-bindings --without-mruby \
    && make -j"$(nproc)" \
    && make install \
    && /opt/nghttp2/bin/nghttpx --version

FROM debian@sha256:b6e2a152f22a40ff69d92cb397223c906017e1391a73c952b588e51af8883bf8

RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        ca-certificates libcares2 libev4t64 libjemalloc2 python3 zlib1g \
    && rm -rf /var/lib/apt/lists/*
COPY --from=builder /opt/aws-lc /opt/aws-lc
COPY --from=builder /opt/nghttp3 /opt/nghttp3
COPY --from=builder /opt/ngtcp2 /opt/ngtcp2
COPY --from=builder /opt/nghttp2 /opt/nghttp2
COPY docker/backend.py /usr/local/lib/protocol-lab/backend.py
COPY docker/entrypoint.sh /usr/local/bin/protocol-lab-nghttpx-entrypoint
COPY certs/leaf.pem /certs/leaf.pem
COPY certs/leaf-key.pem /certs/leaf-key.pem
COPY --from=builder /src/nghttp2/COPYING /licenses/nghttp2-COPYING.txt
COPY --from=builder /src/ngtcp2/COPYING /licenses/ngtcp2-COPYING.txt
COPY --from=builder /src/nghttp3/COPYING /licenses/nghttp3-COPYING.txt
COPY --from=builder /src/aws-lc/LICENSE /licenses/aws-lc-LICENSE.txt
ENV PATH=/opt/nghttp2/bin:$PATH \
    LD_LIBRARY_PATH=/opt/nghttp2/lib:/opt/ngtcp2/lib:/opt/nghttp3/lib:/opt/aws-lc/lib
RUN chmod 0555 /usr/local/bin/protocol-lab-nghttpx-entrypoint /usr/local/lib/protocol-lab/backend.py \
    && chmod 0400 /certs/leaf-key.pem \
    && nghttpx --version
EXPOSE 4433/udp
ENTRYPOINT ["/usr/local/bin/protocol-lab-nghttpx-entrypoint"]
