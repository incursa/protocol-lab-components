FROM python:3.12-slim
ARG AIOQUIC_VERSION=1.3.0
RUN python -m pip install --no-cache-dir "aioquic==${AIOQUIC_VERSION}"
COPY docker/server.py /usr/local/bin/aioquic-webtransport-server
COPY certs /certs
ENTRYPOINT ["python", "/usr/local/bin/aioquic-webtransport-server"]
EXPOSE 4433/udp
