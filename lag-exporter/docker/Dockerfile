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

# Remove old okhttp jars (3.x)
RUN rm -f /opt/docker/lib/com.squareup.okhttp3.okhttp-3.*.jar \
    && rm -f /opt/docker/lib/okio-*.jar

# Download replacement jars (okhttp 4.9.2 + okio 2.8.0)
RUN curl -sSL -o /opt/docker/lib/okhttp-4.9.2.jar \
      https://repo1.maven.org/maven2/com/squareup/okhttp3/okhttp/4.9.2/okhttp-4.9.2.jar \
    && curl -sSL -o /opt/docker/lib/okio-2.8.0.jar \
      https://repo1.maven.org/maven2/com/squareup/okio/okio/2.8.0/okio-2.8.0.jar

RUN chmod -R 777 ${LAG_EXPORTER_HOME}

COPY docker/entrypoint.sh /opt/docker/bin/entrypoint.sh

RUN chmod +x /opt/docker/bin/entrypoint.sh && chmod +x /opt/docker/bin/kafka-lag-exporter

USER 1001

ENTRYPOINT ["/opt/docker/bin/entrypoint.sh"]