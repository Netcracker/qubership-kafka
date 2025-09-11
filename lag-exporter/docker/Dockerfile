FROM seglo/kafka-lag-exporter:0.8.2 as builder

FROM eclipse-temurin:21-jdk-alpine-3.22

COPY --from=builder /opt/docker/bin /opt/docker/bin
COPY --from=builder /opt/docker/lib /opt/docker/lib

ENV LAG_EXPORTER_HOME=/opt/docker

COPY docker/logback.xml /opt/docker/conf/logback.xml

USER 0

RUN set -x \
    && apk add --update --no-cache \
        bash openssl \
    && rm -rf /var/cache/apk/*

# Upgrade all tools to avoid vulnerabilities
RUN set -x && apk upgrade --no-cache --available

RUN chmod -R 777 ${LAG_EXPORTER_HOME}

COPY docker/entrypoint.sh /opt/docker/bin/entrypoint.sh

RUN chmod +x /opt/docker/bin/entrypoint.sh && chmod +x /opt/docker/bin/kafka-lag-exporter

USER 1001

ENTRYPOINT ["/opt/docker/bin/entrypoint.sh"]