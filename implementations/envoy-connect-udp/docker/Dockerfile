ARG GO_VERSION=1.25
FROM golang:${GO_VERSION}-bookworm AS echo-build
WORKDIR /src
COPY source/udp-echo/go.mod ./go.mod
COPY source/udp-echo/main.go ./main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -buildvcs=false -trimpath -ldflags "-s -w" -o /out/udp-echo .

FROM envoyproxy/envoy:v1.38.3@sha256:5f7c43e1147412fdb3af578c651c67478a3df818eae89d2261e707e06c209cdb
COPY --from=echo-build /out/udp-echo /usr/local/bin/udp-echo
COPY docker/envoy.yaml /etc/envoy/envoy.yaml
COPY docker/run.sh /usr/local/bin/protocol-lab-run
COPY certs /certs
RUN chmod 0755 /usr/local/bin/protocol-lab-run /usr/local/bin/udp-echo
EXPOSE 4443/udp
ENTRYPOINT ["/usr/local/bin/protocol-lab-run"]
