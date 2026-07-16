FROM rust@sha256:af306cfa71d987911a781c37b59d7d67d934f49684058f96cf72079c3626bfe0 AS build

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        clang=1:14.0-55.7~deb12u1 \
        cmake=3.25.1-1 \
        libclang-dev=1:14.0-55.7~deb12u1 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src
COPY implementations/quiche-raw/source/Cargo.toml ./Cargo.toml
COPY implementations/quiche-raw/source/Cargo.lock ./Cargo.lock
COPY implementations/quiche-raw/source/src ./src
RUN cargo test --locked \
    && cargo build --release --locked \
    && /src/target/release/protocol-lab-quiche-raw --self-test

FROM debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818
COPY --from=build /src/target/release/protocol-lab-quiche-raw /usr/local/bin/quiche-raw
ENTRYPOINT ["/usr/local/bin/quiche-raw"]
