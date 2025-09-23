FROM eclipse-temurin:21-jdk-alpine-3.22

ENV KAFKA_HOME=/opt/kafka \
    KAFKA_DATA_DIRS=/var/opt/kafka/data \
    KAFKA_LOG_DIRS=/opt/kafka/logs \
    KAFKA_CTL_CONFIG=/opt/kafka/bin/kafkactl.yml \
    KCAT_CONFIG=/opt/kafka/bin/kcat.properties

WORKDIR /

# Install misc tools
RUN set -x \
    && apk add --update --no-cache \
        bash \
        shadow \
        netcat-openbsd \
        jq \
        tini \
        rsync \
        curl \
        grep \
        ca-certificates \
        wget \
        jattach \
        kcat \
        iputils \
        util-linux \
        apk-tools \
        ncdu \
        coreutils \
        openssl \
    && ln -sf /usr/bin/kcat /usr/bin/kafkacat \
    && rm -rf /var/cache/apk/*

# Add unpriviliged user
RUN set -x \
    && groupadd -r kafka --gid=1000 \
    && useradd -s /bin/bash -r -g kafka --uid=1000 kafka \
    && usermod -a -G 0 kafka

ARG KAFKA_VERSION=3.9.1
ARG DISTRO_NAME=kafka_2.12-${KAFKA_VERSION}
# Download Apache Kafka and install
RUN set -x \
    && export DISTR_DIR="$(mktemp -d)" \
    && wget \
        -nv \
        -O ${DISTR_DIR}/${DISTRO_NAME}.tgz \
        "https://downloads.apache.org/kafka/${KAFKA_VERSION}/${DISTRO_NAME}.tgz" \
    && tar -zxf ${DISTR_DIR}/${DISTRO_NAME}.tgz -C ${DISTR_DIR} \
    && mkdir -p ${KAFKA_HOME} \
    && mv ${DISTR_DIR}/${DISTRO_NAME}/* ${KAFKA_HOME} \
    && rm -rf ${DISTR_DIR}

ARG TMP_DIR="/tmp"
ARG TARGETARCH
# Download and install async-profiler
RUN set -x \
    && if [ "${TARGETARCH}" = "amd64" ]; then export AP_ARCH="x64"; \
       elif [ "${TARGETARCH}" = "arm64" ]; then export AP_ARCH="arm64"; \
       else echo "Unsupported architecture: ${TARGETARCH}" && exit 1; fi \
    && wget \
        --no-check-certificate \
        -nv \
        -O ${TMP_DIR}/async-profiler-3.0-linux-${AP_ARCH}.tar.gz \
        "https://github.com/async-profiler/async-profiler/releases/download/v3.0/async-profiler-3.0-linux-${AP_ARCH}.tar.gz" \
    && tar -xzf /tmp/async-profiler-3.0-linux-${AP_ARCH}.tar.gz -C /opt/ \
    && ln -s /opt/async-profiler-3.0-linux-${AP_ARCH}/profiler.sh /usr/local/bin/profiler \
    && rm -f ${TMP_DIR}/async-profiler-3.0-linux-${AP_ARCH}.tar.gz

# Download kafkactl
ARG KAFKACTL_VERSION="5.12.1"

RUN set -x \
    && wget \
        -nv \
        -O /tmp/kafkactl.tar.gz \
        "https://github.com/deviceinsight/kafkactl/releases/download/v${KAFKACTL_VERSION}/kafkactl_${KAFKACTL_VERSION}_linux_${TARGETARCH}.tar.gz" \
    && tar -zxvf /tmp/kafkactl.tar.gz -C /usr/bin \
    && chmod +x /usr/bin/kafkactl \
    && rm -f /tmp/kafkactl.tar.gz

#Download Cruise Control Metric Reporter
ENV CC_VERSION=2.5.141
RUN set -x \
    && wget \
        -nv \
        -O ${KAFKA_HOME}/libs/cruise-control-metrics-reporter-${CC_VERSION}.jar \
        "https://linkedin.jfrog.io/artifactory/cruise-control/com/linkedin/cruisecontrol/cruise-control-metrics-reporter/${CC_VERSION}/cruise-control-metrics-reporter-${CC_VERSION}.jar"

ARG PROMETHEUS_JMX_EXPORTER_VERSION="1.1.0"
# Download jmx_prometheus_javaagent
RUN set -x \
    && wget \
        --no-check-certificate \
        -nv \
        -O ${KAFKA_HOME}/libs/jmx_prometheus_javaagent-${PROMETHEUS_JMX_EXPORTER_VERSION}.jar \
        "https://github.com/prometheus/jmx_exporter/releases/download/${PROMETHEUS_JMX_EXPORTER_VERSION}/jmx_prometheus_javaagent-${PROMETHEUS_JMX_EXPORTER_VERSION}.jar"


ARG JERSEY_VERSION="2.39.1"
ARG JAXB_VERSION="2.3.0"
ARG KAFKA_SECURITY_OAUTH_VERSION="0.11.0-3.9.1"
# Download OAuth2 dependencies
RUN set -x \
    && wget -nv -O ${KAFKA_HOME}/libs/jersey-media-json-jackson-${JERSEY_VERSION}.jar \
        "https://repo1.maven.org/maven2/org/glassfish/jersey/media/jersey-media-json-jackson/${JERSEY_VERSION}/jersey-media-json-jackson-${JERSEY_VERSION}.jar" \
    && wget -nv -O ${KAFKA_HOME}/libs/jersey-entity-filtering-${JERSEY_VERSION}.jar \
        "https://repo1.maven.org/maven2/org/glassfish/jersey/ext/jersey-entity-filtering/${JERSEY_VERSION}/jersey-entity-filtering-${JERSEY_VERSION}.jar" \
    && wget -nv -O ${KAFKA_HOME}/libs/nimbus-jose-jwt-9.37.3.jar \
        "https://repo1.maven.org/maven2/com/nimbusds/nimbus-jose-jwt/9.37.3/nimbus-jose-jwt-9.37.3.jar" \
    && wget -nv -O ${KAFKA_HOME}/libs/jcip-annotations-1.0-1.jar \
        "https://repo1.maven.org/maven2/net/jcip/jcip-annotations/1.0/jcip-annotations-1.0.jar" \
    && wget -nv -O ${KAFKA_HOME}/libs/json-smart-2.4.11.jar \
        "https://repo1.maven.org/maven2/net/minidev/json-smart/2.4.11/json-smart-2.4.11.jar" \
    && wget -nv -O ${KAFKA_HOME}/libs/accessors-smart-2.4.11.jar \
        "https://repo1.maven.org/maven2/net/minidev/accessors-smart/2.4.11/accessors-smart-2.4.11.jar" \
    && wget -nv -O ${KAFKA_HOME}/libs/asm-9.1.jar \
        "https://repo1.maven.org/maven2/org/ow2/asm/asm/9.1/asm-9.1.jar" \
    && wget -nv -O ${KAFKA_HOME}/libs/jaxb-core-${JAXB_VERSION}.jar \
        "https://repo1.maven.org/maven2/com/sun/xml/bind/jaxb-core/${JAXB_VERSION}/jaxb-core-${JAXB_VERSION}.jar" \
    && wget -nv -O ${KAFKA_HOME}/libs/jaxb-impl-${JAXB_VERSION}.jar \
        "https://repo1.maven.org/maven2/com/sun/xml/bind/jaxb-impl/${JAXB_VERSION}/jaxb-impl-${JAXB_VERSION}.jar" \
    && wget -nv -O ${KAFKA_HOME}/libs/guava-32.0.1-jre.jar \
        "https://repo1.maven.org/maven2/com/google/guava/guava/32.0.1-jre/guava-32.0.1-jre.jar"

RUN rm -f ${KAFKA_HOME}/libs/commons-beanutils-1.9.4.jar \
    && wget -nv -O ${KAFKA_HOME}/libs/commons-beanutils-1.11.0.jar \
       "https://repo1.maven.org/maven2/commons-beanutils/commons-beanutils/1.11.0/commons-beanutils-1.11.0.jar"

# Kafka Security OAuth dependencies
RUN set -x \
    && wget -nv -O ${KAFKA_HOME}/libs/kafka-security-oauth-server-${KAFKA_SECURITY_OAUTH_VERSION}.jar \
        "https://repo1.maven.org/maven2/org/qubership/kafka-security/kafka-security-oauth-server/${KAFKA_SECURITY_OAUTH_VERSION}/kafka-security-oauth-server-${KAFKA_SECURITY_OAUTH_VERSION}.jar" \
    && wget -nv -O ${KAFKA_HOME}/libs/kafka-security-oauth-client-${KAFKA_SECURITY_OAUTH_VERSION}.jar \
        "https://repo1.maven.org/maven2/org/qubership/kafka-security/kafka-security-oauth-client/${KAFKA_SECURITY_OAUTH_VERSION}/kafka-security-oauth-client-${KAFKA_SECURITY_OAUTH_VERSION}.jar"

# Copy config and script files
COPY docker/config/ ${KAFKA_HOME}/config
COPY docker/kafka-health.sh ${KAFKA_HOME}/bin
COPY docker/get-kraft-migration-status.sh ${KAFKA_HOME}/bin
COPY docker/get-cluster-id.sh ${KAFKA_HOME}/bin
COPY docker/kafka-partitions.sh ${KAFKA_HOME}/bin
COPY docker/kafka-consumer-group-checker.sh ${KAFKA_HOME}/bin
COPY docker/kafka-partition-logs.sh ${KAFKA_HOME}/bin
COPY docker/docker-entrypoint.sh /

# Adapt grants
RUN mkdir -p "${KAFKA_HOME}" \
    && mkdir -p "${KAFKA_DATA_DIRS}" \
    && mkdir -p "${KAFKA_LOG_DIRS}" \
    && chgrp -R 0 "${KAFKA_HOME}" \
    && chgrp -R 0 "${KAFKA_DATA_DIRS}" \
    && chgrp -R 0 "${KAFKA_LOG_DIRS}" \
    && chmod -R g+rw "${KAFKA_HOME}" \
    && chmod -R g+rw "${KAFKA_DATA_DIRS}" \
    && chmod -R g+rw "${KAFKA_LOG_DIRS}" \
    && find "${KAFKA_HOME}" -type d -exec chmod g+x {} + \
    && find "${KAFKA_DATA_DIRS}" -type d -exec chmod g+x {} + \
    && find "${KAFKA_LOG_DIRS}" -type d -exec chmod g+x {} + \
    && chmod +x ${KAFKA_HOME}/bin/*.sh \
    && chmod +x /docker-entrypoint.sh

# Upgrade all tools to avoid vulnerabilities
RUN set -x && apk upgrade --no-cache --available

# Upgrade tools from edge to avoid vulnerabilities
RUN echo "https://dl-cdn.alpinelinux.org/alpine/edge/main" > /etc/apk/repositories  \
    && echo "https://dl-cdn.alpinelinux.org/alpine/edge/community" >> /etc/apk/repositories \
    && apk add --update --upgrade --no-cache \
        pcre2

USER 1000:0
WORKDIR ${KAFKA_HOME}

# Expose the ports and set up volumes for the data and logs directories
EXPOSE 9092 8081
VOLUME ["${KAFKA_DATA_DIRS}", "${KAFKA_LOG_DIRS}"]

ENTRYPOINT ["/sbin/tini" ,"--", "/docker-entrypoint.sh"]
CMD ["start"]