FROM golang@sha256:ef8c5c733079ac219c77edab604c425d748c740d8699530ea6aced9de79aea40 AS build
WORKDIR /src
COPY helper/go.mod helper/main.go ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w -buildid=" -o /out/plab-unbound-fixture .

FROM mvance/unbound@sha256:76906da36d1806f3387338f15dcf8b357c51ce6897fb6450d6ce010460927e90
USER root
COPY --from=build /out/plab-unbound-fixture /usr/local/bin/plab-unbound-fixture
COPY unbound.conf start.sh /opt/unbound/etc/unbound/plab/
COPY certs/ /opt/unbound/etc/unbound/plab/
RUN chown -R _unbound:_unbound /opt/unbound/etc/unbound/plab \
    && chmod 0755 /usr/local/bin/plab-unbound-fixture /opt/unbound/etc/unbound/plab/start.sh \
    && chmod 0600 /opt/unbound/etc/unbound/plab/leaf-key.pem \
    && /opt/unbound/sbin/unbound-checkconf /opt/unbound/etc/unbound/plab/unbound.conf
EXPOSE 443/tcp 444/tcp
ENTRYPOINT ["/opt/unbound/etc/unbound/plab/start.sh"]
