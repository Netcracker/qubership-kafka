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
    # Upgrade Python tooling and pin known-vulnerable packages
    # (reported by scanners for this image).
    && pip3 install --no-cache-dir --upgrade pip \
    && pip3 install --no-cache-dir "setuptools==80.10.0" "jaraco.context==6.1.0" "jaraco.text==6.1.0" "urllib3==2.6.3" \
    && pip3 install -r ${KAFKA_HOME}/requirements.txt \
    && rm -rf /var/cache/apk/*

# Upgrade all tools to avoid vulnerabilities
RUN set -x && apk upgrade --no-cache --available

RUN chmod +x ${KAFKA_HOME}/scripts/*.py

WORKDIR ${KAFKA_HOME}
USER 1000:0
VOLUME ["${KAFKA_BACKUP}"]
