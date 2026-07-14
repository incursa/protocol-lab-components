FROM alpine:3.22.1@sha256:4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1

ARG WEBSOCAT_VERSION=1.14.1
ARG WEBSOCAT_SHA256=66f8dd3a0394761556339117f8bb5123bddefd44e087af2a72ec22b0bd08d514
ADD https://github.com/vi/websocat/releases/download/v${WEBSOCAT_VERSION}/websocat.x86_64-unknown-linux-musl /usr/local/bin/websocat
RUN echo "${WEBSOCAT_SHA256}  /usr/local/bin/websocat" | sha256sum -c - \
    && chmod 0755 /usr/local/bin/websocat

EXPOSE 18081
ENTRYPOINT ["/usr/local/bin/websocat", "--close-status-code", "1000", "ws-l:0.0.0.0:18081", "mirror:"]
