FROM eclipse-temurin:21-jdk-alpine-3.23 AS builder

ENV CC_VERSION=2.5.146

RUN set -x \
    && apk add --update --no-cache \
      git \
      wget \
      gradle \
      tar

RUN set -x \
    && wget \
        --no-check-certificate \
        -nv \
        -O /tmp/cruise-control.tar.gz \
        "https://github.com/linkedin/cruise-control/archive/refs/tags/${CC_VERSION}.tar.gz" \
    && tar -zxvf /tmp/cruise-control.tar.gz --transform "s/cruise-control-${CC_VERSION}/cruise-control/" -C / \
    && rm -f /tmp/cruise-control.tar.gz

WORKDIR cruise-control

COPY docker/config/build.gradle build.gradle
COPY docker/config/build/build.gradle buildSrc/build.gradle
COPY docker/config/gradle.properties gradle.properties
COPY docker/config/settings.gradle settings.gradle

RUN git init
# Compile and remove leftover directories
RUN gradle jar :cruise-control:jar --stacktrace   \
    && rm -rf cruise-control-core cruise-control-metrics-reporter cruise-control-client

RUN set -x \
    && wget \
        --no-check-certificate \
        -nv \
        -O /tmp/cruise-control-ui.tar.gz \
        "https://github.com/linkedin/cruise-control-ui/releases/download/v0.4.0/cruise-control-ui-0.4.0.tar.gz" \
    && tar -zxvf /tmp/cruise-control-ui.tar.gz -C /cruise-control \
    && rm -f /tmp/cruise-control-ui.tar.gz
# --------------- Final stage ---------------
FROM eclipse-temurin:21-jdk-alpine-3.23

RUN mkdir -p /cruise-control

ENV KAFKA_CTL_CONFIG=/cruise-control/kafkactl.yml

WORKDIR cruise-control

RUN set -x \
    && apk add --update --no-cache \
      openssl \
      bash \
      curl

COPY --from=builder cruise-control .

COPY docker/docker-entrypoint.sh docker-entrypoint.sh

RUN chmod +x docker-entrypoint.sh
RUN chmod -R 777 cruise-control

# Ensure Cruise Control writable for logs
RUN chmod a+rw -R .

ARG TMP_DIR="/tmp"

# Download kafkactl
ARG KAFKACTL_VERSION=5.17.0
ARG TARGETARCH

RUN set -x \
    && TMP_DIR=$(mktemp -d) \
    && wget \
        --no-check-certificate \
        -nv \
        -O ${TMP_DIR}/kafkactl.tar.gz \
        "https://github.com/deviceinsight/kafkactl/releases/download/v${KAFKACTL_VERSION}/kafkactl_${KAFKACTL_VERSION}_linux_${TARGETARCH}.tar.gz" \
    && tar -zxvf ${TMP_DIR}/kafkactl.tar.gz -C /usr/bin \
    && chmod +x /usr/bin/kafkactl \
    && rm -rf ${TMP_DIR}

# Upgrade tools from edge to avoid vulnerabilities
RUN echo "https://dl-cdn.alpinelinux.org/alpine/edge/main" > /etc/apk/repositories  \
    && echo "https://dl-cdn.alpinelinux.org/alpine/edge/community" >> /etc/apk/repositories \
    && apk add --update --upgrade --no-cache \
        pcre2 curl gnutls libcurl jq gnupg

EXPOSE 9090
USER 1001
ENTRYPOINT ["/cruise-control/docker-entrypoint.sh"]
