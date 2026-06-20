ARG GO_VERSION=1.25
ARG QUIC_GO_VERSION=v0.60.0

FROM golang:${GO_VERSION}-bookworm AS build
ARG QUIC_GO_VERSION
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY src ./src
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags "-s -w -X main.quicGoVersion=${QUIC_GO_VERSION}" \
    -o /out/quic-go-http3-server ./src

FROM scratch
COPY --from=build /out/quic-go-http3-server /quic-go-http3-server
EXPOSE 4433/udp
ENTRYPOINT ["/quic-go-http3-server"]
