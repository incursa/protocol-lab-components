FROM technitium/dns-server@sha256:b2b6eeeae5057880c7403da426907ccd83070b5c7a1ecfb12135d98b9f4a0b9e

USER root
RUN apt-get update \
    && apt-get install -y --no-install-recommends curl \
    && rm -rf /var/lib/apt/lists/*
COPY bootstrap.sh /opt/plab/bootstrap.sh
RUN chmod 0755 /opt/plab/bootstrap.sh

ENV DNS_SERVER_DOMAIN=dns.plab.test \
    DNS_SERVER_RECURSION=Deny \
    DNS_SERVER_ENABLE_BLOCKING=false

EXPOSE 53/udp
ENTRYPOINT ["/opt/plab/bootstrap.sh"]
