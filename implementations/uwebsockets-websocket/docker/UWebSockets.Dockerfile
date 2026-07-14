FROM gcc:15.2.0-bookworm@sha256:9ca91b05c7b07d2979f16413e8b2cd6ec8a7c80ffca4121ccab0aeba33f90460 AS build
ARG UWEBSOCKETS_COMMIT=fe7c01a477b688a7743f754fee33bdd78d52ad91
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates git make zlib1g-dev \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /src
RUN git clone https://github.com/uNetworking/uWebSockets.git \
    && cd uWebSockets \
    && git checkout "${UWEBSOCKETS_COMMIT}" \
    && git submodule update --init --depth 1 uSockets
COPY server.cpp /src/uWebSockets/server.cpp
RUN cd /src/uWebSockets/uSockets \
    && gcc -DLIBUS_NO_SSL -std=c11 -Isrc -O3 -c src/*.c src/eventing/*.c src/crypto/*.c \
    && cd .. \
    && g++ -std=c++17 -O3 -static-libstdc++ -static-libgcc -Isrc -IuSockets/src server.cpp uSockets/*.o -lz -pthread -o /tmp/uwebsockets-server

FROM debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818
RUN useradd --system --uid 10001 --no-create-home protocol-lab
COPY --from=build /tmp/uwebsockets-server /usr/local/bin/uwebsockets-server
ENV PLAB_TARGET_PORT=18081
EXPOSE 18081
USER protocol-lab
ENTRYPOINT ["/usr/local/bin/uwebsockets-server"]
