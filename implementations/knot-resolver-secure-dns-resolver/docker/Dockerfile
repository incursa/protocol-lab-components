FROM golang@sha256:ef8c5c733079ac219c77edab604c425d748c740d8699530ea6aced9de79aea40 AS build
WORKDIR /src
COPY helper/go.mod helper/main.go ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w -buildid=" -o /out/plab-knot-resolver-fixture .

FROM docker.io/cznic/knot-resolver@sha256:45368279d00a072858e9994557fd761b5bdd9140a31f214608eec5773c1d4fa0
USER root
COPY --from=build /out/plab-knot-resolver-fixture /usr/local/bin/plab-knot-resolver-fixture
COPY config.yaml start.sh /opt/protocol-lab/knot-resolver/
COPY certs/ /opt/protocol-lab/knot-resolver/
RUN chown -R knot-resolver:knot-resolver /opt/protocol-lab/knot-resolver \
    && chmod 0755 /usr/local/bin/plab-knot-resolver-fixture /opt/protocol-lab/knot-resolver/start.sh \
    && chmod 0600 /opt/protocol-lab/knot-resolver/leaf-key.pem \
    && /usr/bin/kresctl --config /opt/protocol-lab/knot-resolver/config.yaml validate
EXPOSE 853/tcp 443/tcp 444/tcp
ENTRYPOINT ["/opt/protocol-lab/knot-resolver/start.sh"]
CMD []
