FROM python:3.12-slim@sha256:090ba77e2958f6af52a5341f788b50b032dd4ca28377d2893dcf1ecbdfdfe203

ARG AIOQUIC_VERSION=1.3.0
RUN python -m pip install --no-cache-dir "aioquic==${AIOQUIC_VERSION}"
WORKDIR /work
COPY docker/aioquic_http3_websocket_client.py /usr/local/bin/aioquic-http3-websocket-client
COPY tests /work/tests
COPY certs/root.pem /certs/root.pem
COPY third-party/aioquic-LICENSE.txt /licenses/aioquic-LICENSE.txt
RUN chmod +x /usr/local/bin/aioquic-http3-websocket-client
ENTRYPOINT ["python"]
