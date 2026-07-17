ARG GO_VERSION=1.25
FROM golang:${GO_VERSION}-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY src ./src
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -buildvcs=false -trimpath -ldflags "-s -w" -o /out/masque-go-connect-udp ./src

FROM scratch
COPY --from=build /out/masque-go-connect-udp /masque-go-connect-udp
COPY certs /certs
EXPOSE 4443/udp 4433/udp
ENTRYPOINT ["/masque-go-connect-udp"]
