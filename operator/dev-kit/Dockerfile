FROM quay.io/operator-framework/operator-sdk:v1.28@sha256:aac0e09fc691f5f75c7b1dce778b660d5ce401587b8376e16e18e101011bb5ea

RUN microdnf install --nodocs \
    gcc \
    git \
    && microdnf clean all

ENTRYPOINT ["/bin/bash"]