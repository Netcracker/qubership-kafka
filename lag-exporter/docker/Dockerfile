FROM danielqsj/kafka-exporter:latest

## PROBABLY WE SHOULD IMPLEMENT HEALTH CHECKS VIA CONTAINER, NOT THE DOCKERFILE
#HEALTHCHECK --interval=30s --timeout=3s --start-period=20s \
#  CMD wget -qO- http://127.0.0.1:9308/metrics >/dev/null || exit 1


#####

ENV TINI_VERSION=v0.19.0
ENV TINI_SHA256_AMD64=93dcc18adc78c65a028a84799ecf8ad40c936fdfc5f2a57b1acda5a8117fa82c
ENV TINI_SHA256_ARM64=07952557df20bfd2a95f9bef198b445e006171969499a1d361bd9e6f8e5e0e81

RUN set -ex; \
    if [[ "${TARGETOS}/${TARGETARCH}" = "linux/arm64" ]]; then \
        curl -s -L https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini-arm64 -o /usr/bin/tini; \
        echo "${TINI_SHA256_ARM64} */usr/bin/tini" | sha256sum -c; \
        chmod +x /usr/bin/tini; \
    else \
        curl -s -L https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini -o /usr/bin/tini; \
        echo "${TINI_SHA256_AMD64} */usr/bin/tini" | sha256sum -c; \
        chmod +x /usr/bin/tini; \
    fi \


COPY docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

USER 1001

ENTRYPOINT ["/entrypoint.sh"]