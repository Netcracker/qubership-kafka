FROM docker.io/danielqsj/kafka-exporter:v1.9.0@sha256:4150e46b2e9697ac30e4be8de03ec1704b4bdc7037577b83b8d35a32be298c16 AS builder

FROM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b

RUN apk add --no-cache bash python3 tini curl ca-certificates

COPY --from=builder /bin/kafka_exporter /bin/kafka_exporter
RUN chmod +x /bin/kafka_exporter

# Upgrade tools from edge to avoid vulnerabilities
RUN apk add --update --upgrade --no-cache \
        pcre2 curl gnutls libcurl jq musl-utils

COPY docker/config_parser.py /opt/config_parser.py
COPY docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

RUN adduser -D -u 1001 appuser
USER 1001

ENTRYPOINT ["/entrypoint.sh"]
