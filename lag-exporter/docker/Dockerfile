FROM docker.io/danielqsj/kafka-exporter:v1.10.0 AS builder

FROM alpine:3.24

RUN apk add --no-cache bash python3 tini curl ca-certificates

COPY --from=builder /bin/kafka_exporter /bin/kafka_exporter
RUN chmod +x /bin/kafka_exporter

# Upgrade all tools to avoid vulnerabilities
RUN set -x && apk upgrade --no-cache --available

COPY docker/config_parser.py /opt/config_parser.py
COPY docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

RUN adduser -D -u 1001 appuser
USER 1001

ENTRYPOINT ["/entrypoint.sh"]
