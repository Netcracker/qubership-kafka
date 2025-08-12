FROM seglo/kafka-lag-exporter:0.8.2

ENV LAG_EXPORTER_HOME=/opt/docker

COPY docker/logback.xml /opt/docker/conf/logback.xml

USER 0

RUN set -x \
    && apk add --update --no-cache \
        openssl \
    && rm -rf /var/cache/apk/*

# Upgrade all tools to avoid vulnerabilities
RUN set -x && apk upgrade --no-cache --available

RUN chmod -R 777 ${LAG_EXPORTER_HOME}

COPY docker/entrypoint.sh /opt/docker/bin/entrypoint.sh

RUN chmod +x /opt/docker/bin/entrypoint.sh

USER 1001

ENTRYPOINT ["/opt/docker/bin/entrypoint.sh"]