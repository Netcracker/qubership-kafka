FROM ghcr.io/netcracker/qubership-docker-integration-tests:0.1.20

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
    # Upgrade Python tooling and pin known-vulnerable packages
    # (reported by scanners for this image).
    && pip3 install --no-cache-dir --upgrade pip==26.0 \
    && pip3 install --no-cache-dir "setuptools==80.10.2" "jaraco.context==6.1.0" "jaraco.text==4.2.0" "urllib3==2.6.3" "pyasn1==0.6.3" "PyJWT==2.12.0" "wheel==0.46.2" \
    && pip3 install --no-cache-dir -r ${ROBOT_HOME}/requirements.txt \
    && rm -rf /var/cache/apk/*

# Upgrade tools from edge to avoid vulnerabilities
RUN echo "https://dl-cdn.alpinelinux.org/alpine/edge/main" > /etc/apk/repositories  \
&& echo "https://dl-cdn.alpinelinux.org/alpine/edge/community" >> /etc/apk/repositories \
&& apk add --update --upgrade --no-cache \
    musl-utils musl musl-dev libssl3 libcrypto3 nghttp2-libs curl libcurl binutils libuuid

USER 1000:0

# Expose the port
EXPOSE 8080
VOLUME ["${ROBOT_OUTPUT}"]
