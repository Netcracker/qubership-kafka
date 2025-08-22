FROM golang:1.24-alpine3.22 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go

# Install misc tools
RUN set -x \
    && apk add --update --no-cache \
        gcc \
        musl-dev

# Tests
#RUN go test -v ./...
# Build
RUN GOOS=linux GOARCH=amd64 GO111MODULE=on go build -tags musl -a -o additional-metrics main.go

FROM telegraf:1.35-alpine

ENV KAFKA_MONITORING_HOME=/opt/kafka-monitoring \
    KAFKA_CTL_CONFIG=/opt/kafka-monitoring/kafkactl.yml \
    USER=1000

WORKDIR /
COPY --from=builder --chown=${USER_UID} /workspace/additional-metrics .

RUN mkdir -p ${KAFKA_MONITORING_HOME}

COPY docker/config/requirements.txt ${KAFKA_MONITORING_HOME}/requirements.txt
COPY exec-scripts/ ${KAFKA_MONITORING_HOME}/exec-scripts/
COPY docker/docker-entrypoint.sh /docker-entrypoint.sh

RUN chmod +x /docker-entrypoint.sh

RUN set -x \
    && apk add --update --no-cache \
         bash \
         shadow \
         python3 \
         python3-dev \
         tini \
         vim \
         rsync \
         apk-tools \
         build-base \
         librdkafka-dev

# Upgrade all tools to avoid vulnerabilities
RUN set -x && apk upgrade --no-cache --available

RUN rm /usr/lib/python3.12/EXTERNALLY-MANAGED

# Install misc tools
RUN set -x \
    && python3 -m ensurepip \
    && rm -r /usr/lib/python*/ensurepip \
    && pip3 install --upgrade pip setuptools \
    && pip3 install -r ${KAFKA_MONITORING_HOME}/requirements.txt \
    && chmod -R 777 ${KAFKA_MONITORING_HOME} \
    && python3 -m unittest discover -s ${KAFKA_MONITORING_HOME}/exec-scripts -p "test_*.py" \
    && rm -fr ${KAFKA_MONITORING_HOME}/exec-scripts/test_*.py \
    && rm -rf /var/cache/apk/*

ARG TMP_DIR="/tmp"
ARG KAFKACTL_VERSION="3.5.1"
RUN set -x \
    && wget \
        -nv \
        -O /tmp/kafkactl.tar.gz \
        "https://github.com/deviceinsight/kafkactl/releases/download/v${KAFKACTL_VERSION}/kafkactl_${KAFKACTL_VERSION}_linux_amd64.tar.gz" \
    && tar -zxvf /tmp/kafkactl.tar.gz -C /usr/bin \
    && chmod +x /usr/bin/kafkactl \
    && rm -f /tmp/kafkactl.tar.gz
USER 1000:0
WORKDIR ${KAFKA_MONITORING_HOME}

ENTRYPOINT ["/docker-entrypoint.sh"]
