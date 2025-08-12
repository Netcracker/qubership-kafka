FROM telegraf:1.35-alpine

ENV KMM_MONITORING_HOME=/opt/kafka-mirror-maker-monitoring

RUN mkdir -p ${KMM_MONITORING_HOME}

COPY docker/config/requirements.txt ${KMM_MONITORING_HOME}/requirements.txt
COPY exec-scripts/ ${KMM_MONITORING_HOME}/exec-scripts/

# Install misc. tools
RUN set -x \
    && apk add --upgrade --no-cache \
        tini \
        python3 \
        apk-tools

# Upgrade all tools to avoid vulnerabilities
RUN set -x && apk upgrade --no-cache --available

RUN rm /usr/lib/python3.12/EXTERNALLY-MANAGED

RUN set -x \
    && python3 -m ensurepip \
    && rm -r /usr/lib/python*/ensurepip \
    && pip3 install --upgrade pip setuptools \
    && pip3 install -r ${KMM_MONITORING_HOME}/requirements.txt \
    && chmod -R 777 ${KMM_MONITORING_HOME}/exec-scripts

USER 1000:0
WORKDIR ${KMM_MONITORING_HOME}

ENTRYPOINT ["/sbin/tini", "--", "/entrypoint.sh"]
CMD ["telegraf"]
