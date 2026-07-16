FROM python:3.12-slim-bookworm@sha256:d50fb7611f86d04a3b0471b46d7557818d88983fc3136726336b2a4c657aa30b

RUN apt-get update \
    && apt-get install -y --no-install-recommends binutils=2.40-2 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src
COPY implementations/aioquic-raw/source/requirements-build.txt ./requirements-build.txt
RUN python -m pip install --no-cache-dir --requirement requirements-build.txt
COPY implementations/aioquic-raw/source/server.py ./server.py
RUN python server.py --self-test \
    && pyinstaller --clean --onefile --name aioquic-raw server.py \
    && dist/aioquic-raw --self-test \
    && mkdir -p /out/bin \
    && cp dist/aioquic-raw /out/bin/aioquic-raw
