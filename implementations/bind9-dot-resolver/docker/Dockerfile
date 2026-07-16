FROM golang@sha256:ef8c5c733079ac219c77edab604c425d748c740d8699530ea6aced9de79aea40 AS build
WORKDIR /src
COPY helper/go.mod helper/main.go ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w -buildid=" -o /out/plab-bind-control .

FROM internetsystemsconsortium/bind9@sha256:4e2ce9999405e69bfbd1557bfa0208df88f22c444a136a0ec81fb7748be97782
USER root
COPY --from=build /out/plab-bind-control /usr/local/bin/plab-bind-control
COPY resolver.conf authority.conf rndc.conf db.plab.test start.sh /opt/protocol-lab/bind-resolver/
COPY certs/ /opt/protocol-lab/bind-resolver/
RUN chown -R bind:bind /opt/protocol-lab/bind-resolver \
    && chmod 0755 /usr/local/bin/plab-bind-control /opt/protocol-lab/bind-resolver/start.sh \
    && chmod 0600 /opt/protocol-lab/bind-resolver/leaf-key.pem /opt/protocol-lab/bind-resolver/rndc.conf
USER bind
EXPOSE 853/tcp 854/tcp
ENTRYPOINT ["/opt/protocol-lab/bind-resolver/start.sh"]
