FROM busybox@sha256:4279d9b47df4c1b02d80efd8d02cd59b3a8182c1e785a4ff3f6983bee19dc8b0 AS busybox

FROM haproxy@sha256:e271912a69f3114b12aeea3362e32ae02bc8d1e330d748e06da3ae6c0522f73c

USER root
COPY --from=busybox /bin/busybox /bin/busybox
COPY docker/haproxy.cfg /usr/local/etc/haproxy/haproxy.cfg
COPY docker/entrypoint.sh /usr/local/bin/protocol-lab-haproxy-entrypoint
COPY docker/httpd.conf /etc/httpd.conf
COPY docker/fixtures /srv
COPY docker/certs/haproxy.pem /usr/local/etc/haproxy/certs/haproxy.pem
RUN /bin/busybox mkdir -p /srv/bytes \
    && /bin/busybox printf 'Hello, World!' > /srv/plaintext.txt \
    && /bin/busybox head -c 1024 /dev/zero | /bin/busybox tr '\000' 'x' > /srv/bytes/1024 \
    && /bin/busybox head -c 65536 /dev/zero | /bin/busybox tr '\000' 'x' > /srv/bytes/65536 \
    && /bin/busybox chmod 0555 /usr/local/bin/protocol-lab-haproxy-entrypoint \
    && /bin/busybox chmod 0644 /usr/local/etc/haproxy/certs/haproxy.pem /srv/bytes/1024 /srv/bytes/65536

EXPOSE 8080/tcp 8082/tcp 8443/tcp 8443/udp
ENTRYPOINT ["/usr/local/bin/protocol-lab-haproxy-entrypoint"]
