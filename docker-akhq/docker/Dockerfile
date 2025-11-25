FROM eclipse-temurin:17-jdk-ubi10-minimal AS builder
RUN microdnf install -y nodejs npm git && microdnf clean all
# WORKDIR /app

COPY docker/build.gradle /tmp/build.gradle

RUN --mount=type=cache,target=/root/.gradle \
    --mount=type=cache,target=/app \
    cd /app \
    && git init \
    && cp -f /tmp/build.gradle /app \
    && wget -O /tmp/akhq.tar.gz https://github.com/tchiotludo/akhq/archive/refs/tags/0.25.1.tar.gz \
    && tar -xzf /tmp/akhq.tar.gz -C /app --strip-components=1 \
    && ./gradlew shadowJar --no-daemon \
    && cp -rf /app /tmp

RUN cp /tmp/app /

FROM eclipse-temurin:21-jdk-alpine-3.22

USER 0:0

ENV AKHQ_HOME=/app \
    MICRONAUT_CONFIG_FILES=/app/application.yml

COPY docker/docker-entrypoint.sh /
COPY docker/config/streaming.desc ${AKHQ_HOME}/config/streaming.desc
COPY docker/app ${AKHQ_HOME}

# Install misc tools
RUN \
  --mount=type=cache,target=/var/cache/apt,sharing=locked \
  --mount=type=cache,target=/var/lib/apt,sharing=locked \
  rm -f /etc/apt/apt.conf.d/docker-clean && \
  mkdir -p /etc/apt/apt.conf.d/ && \
  chmod 766 /etc/apt/apt.conf.d && \
  echo 'Binary::apt::APT::Keep-Downloaded-Packages "true";' >/etc/apt/apt.conf.d/keep-cache && \
  set -x \
    && apk add --upgrade \
        netcat-openbsd \
        rsync \
        bash \
        wget \
        grep \
        curl \
        procps \
        apk-tools \
        jattach \
        openssl

# Download libraries and adapt grants
# Download libraries and adapt grants
ARG AKHQ_VERSION="0.25.1"
COPY --from=builder /app/build/libs/akhq-*-all.jar /app/akhq.jar
RUN set -x \
    && chmod +x /docker-entrypoint.sh \
    && chmod -R 777 ${AKHQ_HOME}

# Upgrade all tools to avoid vulnerabilities
RUN set -x && apk upgrade --no-cache --available

# Upgrade tools from edge to avoid vulnerabilities
RUN echo "https://dl-cdn.alpinelinux.org/alpine/edge/main" > /etc/apk/repositories  \
    && echo "https://dl-cdn.alpinelinux.org/alpine/edge/community" >> /etc/apk/repositories \
    && apk add --update --upgrade --no-cache \
        pcre2

USER 1000:0
WORKDIR ${AKHQ_HOME}
EXPOSE 8080

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["./akhq"]
