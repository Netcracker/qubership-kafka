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

# Download new jars
ADD https://repo1.maven.org/maven2/com/squareup/okhttp3/okhttp/4.9.2/okhttp-4.9.2.jar \
    /tmp/okhttp-4.9.2.jar
ADD https://repo1.maven.org/maven2/com/squareup/okio/okio/2.8.0/okio-2.8.0.jar \
    /tmp/okio-2.8.0.jar
ADD https://repo1.maven.org/maven2/com/squareup/okhttp3/logging-interceptor/4.9.2/logging-interceptor-4.9.2.jar \
    /tmp/logging-interceptor-4.9.2.jar
ADD https://repo1.maven.org/maven2/org/apache/zookeeper/zookeeper/3.9.1/zookeeper-3.9.1.jar \
    /tmp/zookeeper-3.9.1.jar
ADD https://repo1.maven.org/maven2/org/apache/zookeeper/zookeeper-jute/3.9.1/zookeeper-jute-3.9.1.jar \
    /tmp/zookeeper-jute-3.9.1.jar
ADD https://repo1.maven.org/maven2/org/yaml/snakeyaml/2.0/snakeyaml-2.0.jar \
    /tmp/snakeyaml-2.0.jar

# Replace old with new versions
RUN rm -f /opt/docker/lib/com.squareup.okhttp3.okhttp-3.*.jar \
    && rm -f /opt/docker/lib/com.squareup.okhttp3.logging-interceptor-3.*.jar \
    && rm -f /opt/docker/lib/org.apache.zookeeper.zookeeper-3.*.jar \
    && rm -f /opt/docker/lib/org.apache.zookeeper.zookeeper-jute-3.*.jar \
    && rm -f /opt/docker/lib/org.yaml.snakeyaml-*.jar \
    && mv /tmp/okhttp-4.9.2.jar /opt/docker/lib/ \
    && mv /tmp/okio-2.8.0.jar /opt/docker/lib/ \
    && mv /tmp/logging-interceptor-4.9.2.jar /opt/docker/lib/ \
    && mv /tmp/zookeeper-3.9.1.jar /opt/docker/lib/ \
    && mv /tmp/zookeeper-jute-3.9.1.jar /opt/docker/lib/ \
    && mv /tmp/snakeyaml-2.0.jar /opt/docker/lib/

RUN chmod -R 777 ${LAG_EXPORTER_HOME}

COPY docker/entrypoint.sh /opt/docker/bin/entrypoint.sh

RUN chmod +x /opt/docker/bin/entrypoint.sh && chmod +x /opt/docker/bin/kafka-lag-exporter

USER 1001

ENTRYPOINT ["/opt/docker/bin/entrypoint.sh"]