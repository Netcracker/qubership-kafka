#!/usr/bin/env bash
set -euo pipefail
set +x

: "${KAFKA_EXPORTER_HOME:=/bin}"
export LOG_DIR="${KAFKA_EXPORTER_HOME}"

mapfile -t flags < <(python3 /opt/config_parser.py \
  --conf "/opt/docker/src/application.conf")

args=()
args=("${flags[@]}")

normalize_mechanism() {
  local mech
  mech=$(echo "$1" | tr '[:upper:]' '[:lower:]')
  case "$mech" in
    scram-sha-512) echo "scram-sha512" ;;
    scram-sha-256) echo "scram-sha256" ;;
    *) echo "$mech" ;;
  esac
}

if [[ "${KAFKA_ENABLE_SSL}" == "true" ]]; then
  args+=("--tls.enabled")
  [[ -f /tls/ca.crt ]] && args+=("--tls.ca-file=/tls/ca.crt")
  [[ -f /tls/tls.crt ]] && args+=("--tls.cert-file=/tls/tls.crt")
  [[ -f /tls/tls.key ]] && args+=("--tls.key-file=/tls/tls.key")
fi

if [[ -n "${KAFKA_USER}" && -n "${KAFKA_PASSWORD}" ]]; then
  args+=("--sasl.enabled")
  args+=("--sasl.mechanism=$(normalize_mechanism "$KAFKA_SASL_MECHANISM")")
  args+=("--sasl.username=${KAFKA_USER}")
  args+=("--sasl.password=${KAFKA_PASSWORD}")
fi

exec /sbin/tini -w -e 143 -- "$KAFKA_EXPORTER_HOME/kafka_exporter" "${args[@]}"
