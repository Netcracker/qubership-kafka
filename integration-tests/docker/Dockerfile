FROM ghcr.io/netcracker/qubership-docker-integration-tests:main

ENV ROBOT_OUTPUT=${ROBOT_HOME}/output \
    DISTR_DIR=/tmp/deps \
    SERVICE_CHECKER_SCRIPT=${ROBOT_HOME}/kafka_pods_checker.py \
    SERVICE_CHECKER_SCRIPT_TIMEOUT=500

RUN echo "https://dl-cdn.alpinelinux.org/alpine/v3.23/main" > /etc/apk/repositories && \
    echo "https://dl-cdn.alpinelinux.org/alpine/v3.23/community" >> /etc/apk/repositories

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

# Fix permissions for non-root user (1000:0)
# Added for AWS EKS security policies and S3 adapter scripts.
# When files are copied while USER is root, they get root ownership.
# This fixes ownership for:
# 1. AWS EKS strict security policies that restrict root execution
# 2. S3 upload scripts (inotifywait, s5cmd) from base image that need write access
# 3. Robot Framework to create output files and logs
RUN chown -R 1000:0 ${ROBOT_HOME} && chmod -R 775 ${ROBOT_HOME}

# Switch back to non-root user for AWS EKS security compliance
USER 1000:0

# Expose the port
EXPOSE 8080
VOLUME ["${ROBOT_OUTPUT}"]
