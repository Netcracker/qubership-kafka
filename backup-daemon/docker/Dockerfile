FROM ghcr.io/netcracker/qubership-backup-daemon:main_alpine

ENV KAFKA_HOME=/opt/kafka \
    KAFKA_BACKUP=/opt/kafka/backup-storage

RUN echo "https://dl-cdn.alpinelinux.org/alpine/v3.23/main" > /etc/apk/repositories && \
    echo "https://dl-cdn.alpinelinux.org/alpine/v3.23/community" >> /etc/apk/repositories

COPY backup-daemon.conf /etc/backup-daemon.conf
COPY docker/requirements.txt ${KAFKA_HOME}/requirements.txt
COPY scripts/ ${KAFKA_HOME}/scripts/

RUN set -x \
    && apk add --update --no-cache curl apk-tools \
    && pip3 install setuptools \
    && pip3 install -r ${KAFKA_HOME}/requirements.txt \
    && rm -rf /var/cache/apk/*

# Upgrade all tools to avoid vulnerabilities
RUN set -x && apk upgrade --no-cache --available

RUN chmod +x ${KAFKA_HOME}/scripts/*.py

WORKDIR ${KAFKA_HOME}
USER 1000:0
VOLUME ["${KAFKA_BACKUP}"]
