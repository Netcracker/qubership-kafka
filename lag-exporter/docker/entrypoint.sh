#!/usr/bin/env bash
set -euo pipefail
set +x

: "${KAFKA_EXPORTER_HOME:=/bin}"
export LOG_DIR="${KAFKA_EXPORTER_HOME}"

mapfile -t flags < <(python3 /opt/config_parser.py \
  --conf "${LEGACY_CONF_PATH}" \
  --block kafka_exporter \
  --mode lines \
  --quote none)

args=("${flags[@]}")

if [[ "${KAFKA_ENABLE_SSL}" == "true" ]]; then
  args+=("--tls.enabled")
  [[ -f /tls/ca.crt ]] && args+=("--tls.ca-file=/tls/ca.crt")
  [[ -f /tls/tls.crt ]] && args+=("--tls.cert-file=/tls/tls.crt")
  [[ -f /tls/tls.key ]] && args+=("--tls.key-file=/tls/tls.key")
  [[ -n "${TLS_SERVER_NAME}" ]] && args+=("--tls.server-name=${TLS_SERVER_NAME}")
fi

if [[ -n "${KAFKA_USER}" && -n "${KAFKA_PASSWORD}" ]]; then
  args+=("--sasl.enabled")
  args+=("--sasl.mechanism=${KAFKA_SASL_MECHANISM}")
  args+=("--sasl.username=${KAFKA_USER}")
  args+=("--sasl.password=${KAFKA_PASSWORD}")
fi

set -x

exec /sbin/tini -w -e 143 -- "$KAFKA_EXPORTER_HOME/kafka_exporter" "${args[@]}"
