FROM amazoncorretto:11-alpine AS builder

RUN apk add --no-cache maven git

WORKDIR /build

COPY kafka-mm2-extension/pom.xml .
RUN mvn dependency:go-offline

COPY kafka-mm2-extension /build
RUN mvn clean package

FROM eclipse-temurin:21-jdk-alpine-3.22

ENV KAFKA_HOME=/opt/kafka \
    KAFKA_DATA_DIRS=/var/opt/kafka/data \
    KAFKA_LOG_DIRS=/opt/kafka/logs \
    KAFKA_PLUGINS_DIR=${KAFKA_HOME}/plugins

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
        apk-tools \
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
ARG KAFKA_DISTRO_URL="https://downloads.apache.org/kafka/${KAFKA_VERSION}/${DISTRO_NAME}.tgz"
# Download Apache Kafka and install
RUN set -x \
    && export DISTR_DIR="$(mktemp -d)" \
    && wget \
        --no-check-certificate \
        -nv \
        -O ${DISTR_DIR}/${DISTRO_NAME}.tgz \
        "${KAFKA_DISTRO_URL}" \
    && tar -zxf ${DISTR_DIR}/${DISTRO_NAME}.tgz -C ${DISTR_DIR} \
    && mkdir -p ${KAFKA_HOME} \
    && mv ${DISTR_DIR}/${DISTRO_NAME}/* ${KAFKA_HOME} \
    && rm -rf ${DISTR_DIR}

ARG TMP_DIR="/tmp"
ARG TARGETARCH
## Download and install async-profiler
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

ARG PROMETHEUS_JMX_EXPORTER_VERSION="1.1.0"
# Download jmx_prometheus_javaagent
RUN set -x \
    && wget \
        --no-check-certificate \
        -nv \
        -O ${KAFKA_HOME}/libs/prometheus-jmx-exporter-${PROMETHEUS_JMX_EXPORTER_VERSION}.jar \
        "https://github.com/prometheus/jmx_exporter/releases/download/${PROMETHEUS_JMX_EXPORTER_VERSION}/jmx_prometheus_javaagent-${PROMETHEUS_JMX_EXPORTER_VERSION}.jar"

# Install Kubernetes config provider
ARG KAFKA_KUBERNETES_CONFIG_PROVIDER_VERSION="1.2.0"
RUN set -x \
    && export DISTR_DIR="$(mktemp -d)" \
    && wget \
        --no-check-certificate \
        -nv \
        -O ${DISTR_DIR}/plugin.tar.gz \
        "https://github.com/strimzi/kafka-kubernetes-config-provider/releases/download/${KAFKA_KUBERNETES_CONFIG_PROVIDER_VERSION}/kafka-kubernetes-config-provider-${KAFKA_KUBERNETES_CONFIG_PROVIDER_VERSION}.tar.gz" \
    && tar -xzf ${DISTR_DIR}/plugin.tar.gz -C ${DISTR_DIR} \
    && mkdir -p ${KAFKA_PLUGINS_DIR}/kafka-kubernetes-config-provider \
    && mv ${DISTR_DIR}/kafka-kubernetes-config-provider-${KAFKA_KUBERNETES_CONFIG_PROVIDER_VERSION}/libs/* ${KAFKA_PLUGINS_DIR}/kafka-kubernetes-config-provider \
    && rm -rf ${DISTR_DIR}

# Add Mirror Maker extension
RUN set -x \
    && wget \
        --no-check-certificate \
        -nv \
        -O ${KAFKA_HOME}/libs/mirror-maker-2-extensions-1.2.0.jar \
        "https://repo1.maven.org/maven2/io/strimzi/mirror-maker-2-extensions/1.2.0/mirror-maker-2-extensions-1.2.0.jar"

# Copy kafka mirror maker extension
COPY --from=builder /build/target/kafka-mm2-extension-*.jar ${KAFKA_HOME}/libs

# Copy config and script files
COPY docker/config/ ${KAFKA_HOME}/config
COPY docker/health.sh ${KAFKA_HOME}/bin
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

# Expose the ports and set up volumes for the logs directory
VOLUME ["${KAFKA_LOG_DIRS}"]

ENTRYPOINT ["/sbin/tini" ,"--", "/docker-entrypoint.sh"]
CMD ["start"]
