FROM docker.io/danielqsj/kafka-exporter:v1.9.0@sha256:4150e46b2e9697ac30e4be8de03ec1704b4bdc7037577b83b8d35a32be298c16 AS builder

FROM alpine:3.24@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6

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
