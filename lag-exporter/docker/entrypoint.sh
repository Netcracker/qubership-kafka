#!/usr/bin/env bash
set -euo pipefail
set +x

: "${KAFKA_EXPORTER_HOME:=/bin}"
SECRETS_DIR="${SECRETS_DIR:-/etc/secrets/monitoring-pod-secrets}"
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

resolve_secret_value() {
  local secret_key="$1"
  local env_var_name="$2"
  local secret_path="${SECRETS_DIR}/${secret_key}"
  if [[ -r "${secret_path}" ]]; then
    tr -d '\r' < "${secret_path}"
    return 0
  fi
  printf "%s" "${!env_var_name:-}"
}

kafka_user="$(resolve_secret_value "client_username" "KAFKA_USER")"
kafka_password="$(resolve_secret_value "client_password" "KAFKA_PASSWORD")"

if [[ "${KAFKA_ENABLE_SSL}" == "true" ]]; then
  args+=("--tls.enabled")
  [[ -f /tls/ca.crt ]] && args+=("--tls.ca-file=/tls/ca.crt")
  [[ -f /tls/tls.crt ]] && args+=("--tls.cert-file=/tls/tls.crt")
  [[ -f /tls/tls.key ]] && args+=("--tls.key-file=/tls/tls.key")
fi

if [[ -n "${kafka_user}" && -n "${kafka_password}" ]]; then
  args+=("--sasl.enabled")
  args+=("--sasl.mechanism=$(normalize_mechanism "$KAFKA_SASL_MECHANISM")")
  args+=("--sasl.username=${kafka_user}")
  args+=("--sasl.password=${kafka_password}")
fi

exec /sbin/tini -w -e 143 -- "$KAFKA_EXPORTER_HOME/kafka_exporter" "${args[@]}"
