FROM ghcr.io/netcracker/qubership-docker-integration-tests:0.1.18

ENV ROBOT_OUTPUT=${ROBOT_HOME}/output \
    DISTR_DIR=/tmp/deps \
    SERVICE_CHECKER_SCRIPT=${ROBOT_HOME}/kafka_pods_checker.py \
    SERVICE_CHECKER_SCRIPT_TIMEOUT=500

RUN mkdir -p ${ROBOT_HOME} \
    && mkdir -p ${ROBOT_OUTPUT}

COPY docker/requirements.txt ${ROBOT_HOME}/requirements.txt
COPY docker/kafka_pods_checker.py ${ROBOT_HOME}/kafka_pods_checker.py
COPY robot ${ROBOT_HOME}

# Upgrade all tools to avoid vulnerabilities
RUN set -x && apk upgrade --no-cache --available

RUN set -x \
    && pip3 install -r ${ROBOT_HOME}/requirements.txt \
    && rm -rf /var/cache/apk/*

USER 1000:0

# Expose the port
EXPOSE 8080
VOLUME ["${ROBOT_OUTPUT}"]
