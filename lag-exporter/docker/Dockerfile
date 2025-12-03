FROM docker.io/danielqsj/kafka-exporter:latest AS builder

FROM alpine:3.20

RUN apk add --no-cache bash python3 tini curl ca-certificates

COPY --from=builder /bin/kafka_exporter /bin/kafka_exporter
RUN chmod +x /bin/kafka_exporter

COPY docker/config_parser.py /opt/config_parser.py
COPY docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

RUN adduser -D -u 1001 appuser
USER 1001

ENTRYPOINT ["/entrypoint.sh"]
