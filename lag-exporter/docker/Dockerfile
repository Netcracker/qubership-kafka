FROM docker.io/danielqsj/kafka-exporter:v1.10.0@sha256:346a53c2489a3530790d01f7b96632496d0d58926662f4c35bb65f569db2ab42 AS builder

FROM alpine:3.24@sha256:3f4d6b32e26703c6082839234f43769d5046b3ac7c910033911d1e71d8cd630a

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
