FROM python:3.11-alpine

ENV CRD_INIT_HOME=/opt/crd-init
USER 0

COPY docker/requirements.txt requirements.txt

COPY docker/run.sh ${CRD_INIT_HOME}/run.sh

RUN apk add --update --no-cache bash build-base apk-tools
RUN pip install --no-cache-dir --upgrade pip \
    && pip install --no-cache-dir "setuptools==80.10.1" "jaraco.context==6.1.0" "jaraco.text==6.1.0" \
    && pip install --no-cache-dir -r requirements.txt

# Upgrade base Alpine packages to pull patched versions (e.g., binutils).
RUN apk upgrade --no-cache --available

COPY crds ${CRD_INIT_HOME}/crds
COPY crd-init.py ${CRD_INIT_HOME}/crd-init.py

RUN chmod 777 ${CRD_INIT_HOME}/run.sh
RUN chmod -R 777 ${CRD_INIT_HOME}/crds

WORKDIR ${CRD_INIT_HOME}
USER 1000:0
CMD ["/bin/sh","-ec","${CRD_INIT_HOME}/run.sh"]