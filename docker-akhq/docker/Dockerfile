FROM eclipse-temurin:21-jdk-alpine-3.22

USER 0:0

ENV AKHQ_HOME=/app \
    MICRONAUT_CONFIG_FILES=/app/application.yml

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
        openssl

# Download libraries and adapt grants
# Download libraries and adapt grants
ARG AKHQ_VERSION="0.25.1"
RUN set -x \
    && wget -nv -O ${AKHQ_HOME}/akhq.jar \
        "https://github.com/tchiotludo/akhq/releases/download/${AKHQ_VERSION}/akhq-${AKHQ_VERSION}-all.jar" \
    && chmod +x /docker-entrypoint.sh \
    && chmod -R 777 ${AKHQ_HOME}

# Upgrade all tools to avoid vulnerabilities
RUN set -x && apk upgrade --no-cache --available

USER 1000:0
WORKDIR ${AKHQ_HOME}
EXPOSE 8080

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["./akhq"]
