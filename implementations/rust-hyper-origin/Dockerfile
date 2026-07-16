FROM rust@sha256:9dfaae478ecd298b6b5a039e1f2cc4fc040fc818a2de9aa78fa714dea036574d AS build

WORKDIR /src
RUN apk add --no-cache musl-dev=1.2.5-r12
COPY source/Cargo.toml source/Cargo.lock ./
COPY source/src ./src
RUN cargo build --locked --release

FROM alpine@sha256:4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1
RUN addgroup -S plab && adduser -S -G plab plab
COPY --from=build /src/target/release/rust-hyper-origin /usr/local/bin/rust-hyper-origin
USER plab
ENV PLAB_HTTP_PORT=8080
EXPOSE 8080 8082
ENTRYPOINT ["/usr/local/bin/rust-hyper-origin"]
