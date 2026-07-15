FROM internetsystemsconsortium/bind9@sha256:4e2ce9999405e69bfbd1557bfa0208df88f22c444a136a0ec81fb7748be97782
USER root
COPY named.conf /etc/bind/named.conf
COPY db.plab.test /etc/bind/plab/db.plab.test
COPY certs/ /etc/bind/plab/
RUN chown -R bind:bind /etc/bind/plab && chmod 0600 /etc/bind/plab/leaf-key.pem
USER bind
EXPOSE 443/tcp
ENTRYPOINT ["named"]
CMD ["-g","-c","/etc/bind/named.conf"]
