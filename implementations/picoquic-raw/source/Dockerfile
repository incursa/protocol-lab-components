FROM debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818 AS build

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        build-essential=12.9 \
        ca-certificates=20230311+deb12u1 \
        cmake=3.25.1-1 \
        git=1:2.39.5-0+deb12u3 \
        libssl-dev=3.0.20-1~deb12u2 \
        pkg-config=1.8.1-1 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src
COPY CMakeLists.txt main.c ./
RUN cmake -S . -B build -DCMAKE_BUILD_TYPE=Release \
    && cmake --build build --parallel \
    && ctest --test-dir build --output-on-failure \
    && mkdir -p /out/bin /out/certs /out/lib \
    && cp build/protocol-lab-picoquic-raw /out/bin/picoquic-raw \
    && cp build/certs/cert.pem build/certs/key.pem /out/certs/ \
    && cp /usr/lib/x86_64-linux-gnu/libssl.so.3 /usr/lib/x86_64-linux-gnu/libcrypto.so.3 /out/lib/
