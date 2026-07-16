FROM golang@sha256:ef8c5c733079ac219c77edab604c425d748c740d8699530ea6aced9de79aea40 AS build
WORKDIR /src
COPY helper/go.mod helper/main.go ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w -buildid=" -o /out/plab-bind-control .

FROM internetsystemsconsortium/bind9@sha256:4e2ce9999405e69bfbd1557bfa0208df88f22c444a136a0ec81fb7748be97782
USER root
COPY --from=build --chown=bind:bind --chmod=0755 /out/plab-bind-control /usr/local/bin/plab-bind-control
COPY --chown=bind:bind --chmod=0755 start.sh /etc/bind/plab/start.sh
COPY --chown=bind:bind --chmod=0644 resolver.conf authority.conf db.plab.test /etc/bind/plab/
COPY --chown=bind:bind --chmod=0600 rndc.conf /etc/bind/plab/rndc.conf
COPY --chown=bind:bind --chmod=0644 certs/leaf.pem /etc/bind/plab/leaf.pem
COPY --chown=bind:bind --chmod=0600 certs/leaf-key.pem /etc/bind/plab/leaf-key.pem
USER bind
EXPOSE 853/tcp 854/tcp
ENTRYPOINT ["/etc/bind/plab/start.sh"]
