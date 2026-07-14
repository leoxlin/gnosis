FROM ubuntu:24.04

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates git python3 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY gnosis-real /usr/local/bin/gnosis-real
COPY gnosis /usr/local/bin/gnosis
COPY .gnosis-core .gnosis-core
COPY workspace .

RUN chmod +x /usr/local/bin/gnosis /usr/local/bin/gnosis-real \
    && git init -q \
    && git config user.name "gnosis integration" \
    && git config user.email "integration@example.invalid" \
    && git add . \
    && git commit -qm fixture
