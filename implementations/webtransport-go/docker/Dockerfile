ARG GO_VERSION=1.25
FROM golang:${GO_VERSION}-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY src ./src
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -buildvcs=false -trimpath -ldflags "-s -w" -o /out/webtransport-go-server ./src

FROM scratch
COPY --from=build /out/webtransport-go-server /webtransport-go-server
COPY certs /certs
EXPOSE 4433/udp
ENTRYPOINT ["/webtransport-go-server"]
