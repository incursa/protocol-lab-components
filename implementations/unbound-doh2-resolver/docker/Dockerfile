FROM golang@sha256:ef8c5c733079ac219c77edab604c425d748c740d8699530ea6aced9de79aea40 AS build
WORKDIR /src
COPY helper/go.mod helper/main.go ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w -buildid=" -o /out/plab-unbound-fixture .

FROM mvance/unbound@sha256:76906da36d1806f3387338f15dcf8b357c51ce6897fb6450d6ce010460927e90
USER root
COPY --from=build --chown=_unbound:_unbound --chmod=0755 /out/plab-unbound-fixture /usr/local/bin/plab-unbound-fixture
COPY --chown=_unbound:_unbound --chmod=0755 start.sh /opt/unbound/etc/unbound/plab/start.sh
COPY --chown=_unbound:_unbound --chmod=0644 unbound.conf /opt/unbound/etc/unbound/plab/unbound.conf
COPY --chown=_unbound:_unbound --chmod=0644 certs/leaf.pem /opt/unbound/etc/unbound/plab/leaf.pem
COPY --chown=_unbound:_unbound --chmod=0600 certs/leaf-key.pem /opt/unbound/etc/unbound/plab/leaf-key.pem
RUN /opt/unbound/sbin/unbound-checkconf /opt/unbound/etc/unbound/plab/unbound.conf
EXPOSE 443/tcp 444/tcp
ENTRYPOINT ["/opt/unbound/etc/unbound/plab/start.sh"]
