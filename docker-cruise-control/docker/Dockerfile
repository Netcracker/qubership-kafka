FROM eclipse-temurin:21-jdk-alpine-3.22 AS builder

ENV CC_VERSION=2.5.141

RUN \
  --mount=type=cache,target=/var/cache/apt,sharing=locked \
  --mount=type=cache,target=/var/lib/apt,sharing=locked \
  rm -f /etc/apt/apt.conf.d/docker-clean && \
  mkdir -p /etc/apt/apt.conf.d/ && \
  chmod 766 /etc/apt/apt.conf.d && \
  echo 'Binary::apt::APT::Keep-Downloaded-Packages "true";' >/etc/apt/apt.conf.d/keep-cache && \
  set -x \
    && apk add --update \
      git \
      wget \
      gradle \
      tar

COPY docker/config/build.gradle /tmp/build.gradle
COPY docker/config/build/build.gradle /tmp/buildSrc/build.gradle
COPY docker/config/settings.gradle /tmp/settings.gradle

RUN --mount=type=cache,target=/root/.gradle \
    --mount=type=cache,target=/cruise-control \
    set -x \
    && wget \
        --no-check-certificate \
        -nv \
        -O /tmp/cruise-control.tar.gz \
        "https://github.com/linkedin/cruise-control/archive/refs/tags/${CC_VERSION}.tar.gz" \
    && tar -zxvf /tmp/cruise-control.tar.gz --transform "s/cruise-control-${CC_VERSION}/cruise-control/" -C / \
    && rm -f /tmp/cruise-control.tar.gz \
    && cd cruise-control \
    && git init && gradle jar :cruise-control:jar --stacktrace \
    && wget \
        --no-check-certificate \
        -nv \
        -O /tmp/cruise-control-ui.tar.gz \
        "https://github.com/linkedin/cruise-control-ui/releases/download/v0.4.0/cruise-control-ui-0.4.0.tar.gz" \
    && tar -zxvf /tmp/cruise-control-ui.tar.gz -C /cruise-control \
    && rm -f /tmp/cruise-control-ui.tar.gz

# --------------- Final stage ---------------
FROM eclipse-temurin:21-jdk-alpine-3.22

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
RUN cd /cruise-control && \
    rm -rf cruise-control-core cruise-control-metrics-reporter cruise-control-client && \
    chmod -R 777 cruise-control

# Ensure Cruise Control writable for logs
RUN chmod a+rw -R .

ARG TMP_DIR="/tmp"

# Download kafkactl
ARG KAFKACTL_VERSION=5.12.1
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

EXPOSE 9090
USER 1001
ENTRYPOINT ["/cruise-control/docker-entrypoint.sh"]
