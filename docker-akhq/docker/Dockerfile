FROM eclipse-temurin:25-jdk-alpine-3.23

USER 0:0

ENV AKHQ_HOME=/app \
    MICRONAUT_CONFIG_FILES=/app/application.yml \
    KAFKA_CTL_CONFIG=/tmp/akhq/kafkactl.yml \
    KCAT_CONFIG=/tmp/akhq/kcat.properties \
    USER=1000

COPY docker/docker-entrypoint.sh /
COPY docker/config/streaming.desc ${AKHQ_HOME}/config/streaming.desc
COPY docker/app ${AKHQ_HOME}

# Install misc tools
RUN set -x \
    && apk add --upgrade --no-cache \
        netcat-openbsd \
        rsync \
        bash \
        wget \
        grep \
        curl \
        procps \
        apk-tools \
        jattach \
        openssl \
        kcat \
    && ln -sf /usr/bin/kcat /usr/bin/kafkacat

# Download libraries and adapt grants
ARG AKHQ_VERSION="0.27.1"
RUN set -x \
    && wget -nv -O ${AKHQ_HOME}/akhq.jar \
        "https://github.com/tchiotludo/akhq/releases/download/${AKHQ_VERSION}/akhq-${AKHQ_VERSION}-all.jar" \
    && chmod +x /docker-entrypoint.sh \
    && chmod -R 777 ${AKHQ_HOME}

# Upgrade all tools to avoid vulnerabilities
RUN set -x && apk upgrade --no-cache --available

# Upgrade tools from edge to avoid vulnerabilities
RUN  apk add --update --upgrade --no-cache \
        pcre2 curl gnutls libcurl jq

ARG KAFKACTL_VERSION="5.19.0"
ARG TARGETARCH
RUN set -x \
    && wget \
        -nv \
        -O /tmp/kafkactl.tar.gz \
        "https://github.com/deviceinsight/kafkactl/releases/download/v${KAFKACTL_VERSION}/kafkactl_${KAFKACTL_VERSION}_linux_${TARGETARCH}.tar.gz" \
    && tar -zxvf /tmp/kafkactl.tar.gz -C /usr/bin \
    && chmod +x /usr/bin/kafkactl \
    && rm -f /tmp/kafkactl.tar.gz

USER 1000:0
WORKDIR ${AKHQ_HOME}
EXPOSE 8080

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["./akhq"]
