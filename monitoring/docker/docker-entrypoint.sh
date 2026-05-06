#!/bin/bash

# Initialise writable runtime directories under /tmp so the container can run
# with readOnlyRootFilesystem: true.
mkdir -p /tmp/monitoring
export KAFKA_CTL_CONFIG=/tmp/monitoring/kafkactl.yml

# Exit immediately if a *pipeline* returns a non-zero status. (Add -x for command tracing)
set -e
if [[ "$DEBUG" == true ]]; then
  set -x
  printenv
fi

: ${KAFKA_SERVICE_NAME:="kafka"}
: ${KAFKA_TOTAL_BROKERS_COUNT:=-1}
: ${KAFKA_CLIENT_PORT:=9092}
: ${KAFKA_PROMETHEUS_PORT:=8080}

# Set environment variables used in telegraf.conf
export KAFKA_USER=${KAFKA_USER}
export KAFKA_PASSWORD=${KAFKA_PASSWORD}
export SM_DB_USERNAME=${SM_DB_USERNAME}
export SM_DB_PASSWORD=${SM_DB_PASSWORD}
export PROMETHEUS_USERNAME=${PROMETHEUS_USERNAME}
export PROMETHEUS_PASSWORD=${PROMETHEUS_PASSWORD}

#Generates addresses with specified port.
#
#$1 - port
function generate_addresses() {
  declare -a addresses
  for (( broker_id=1; broker_id <= ${KAFKA_TOTAL_BROKERS_COUNT}; broker_id++ )); do
    kafka_host="$KAFKA_SERVICE_NAME-$broker_id.$OS_PROJECT"
    addresses+=("$kafka_host:$1")
  done
  OLD_IFS=$IFS; IFS=","
  local output=$(echo "${addresses[*]}")
  IFS=${OLD_IFS}
  echo ${output}
}

#Validate variable name that contains addresses.
#
#$1 - variable name that contains addresses
function validate_addresses() {
  local variable_name=${1}
  OLD_IFS=$IFS; IFS=","
  read -a addresses <<< "${!variable_name}"
  IFS=${OLD_IFS}
  if [[ "${#addresses[@]}" -ne "$KAFKA_TOTAL_BROKERS_COUNT" ]]; then
    echo >&2 "Error: number of addresses in $variable_name variable must be equal to $KAFKA_TOTAL_BROKERS_COUNT!"
    exit 1
  fi
}

#Converts comma separated Prometheus addresses to urls.
#
#$1 - comma separated Prometheus addresses
function convert_to_prometheus_urls() {
  OLD_IFS=$IFS; IFS=","
  read -a addresses <<< "$1"
  IFS=${OLD_IFS}
  declare -a quoted_addresses
  for address in "${addresses[@]}"; do
    quoted_addresses+=("'http://$address/metrics'")
  done
  OLD_IFS=$IFS; IFS=","
  local output=$(echo "${quoted_addresses[*]}")
  IFS=${OLD_IFS}
  echo ${output}
}

function enrich_kafkactl_yml_with_ssl_configs() {
  if [[ "${KAFKA_ENABLE_SSL}" == "true" && -f "/tls/ca.crt" ]]; then
    cat >> ${KAFKA_CTL_CONFIG} << EOL
    tls:
      enabled: true
EOL
    if [[ -f "/tls/ca.crt" ]]; then
      cat >> ${KAFKA_CTL_CONFIG} << EOL
      ca: /tls/ca.crt
EOL
    fi
    if [[ -f "/tls/tls.crt" && -f "/tls/tls.key" ]]; then
      cat >> ${KAFKA_CTL_CONFIG} << EOL
      cert: /tls/tls.crt
      certKey: /tls/tls.key
EOL
    fi
  fi
}

function prepare_secured_config_files() {
  cat > ${KAFKA_CTL_CONFIG} << EOL
current-context: default
contexts:
  default:
    brokers: [${KAFKA_ADDRESSES}]
    sasl:
      enabled: true
      mechanism: scram-sha512
      username: ${KAFKA_USER}
      password: ${KAFKA_PASSWORD}
EOL
}

function prepare_unsecured_config_files() {
  cat > ${KAFKA_CTL_CONFIG} << EOL
current-context: default
contexts:
  default:
    brokers: [${KAFKA_ADDRESSES}]
    sasl:
      enabled: false
EOL
}

if [[ "$KAFKA_TOTAL_BROKERS_COUNT" -le 0 ]]; then
  echo >&2 "Error: KAFKA_TOTAL_BROKERS_COUNT variable must be greater than zero!"
  exit 1
fi

if [[ -z "$KAFKA_ADDRESSES" ]]; then
  export KAFKA_ADDRESSES=$(generate_addresses "$KAFKA_CLIENT_PORT")
fi

if [[ -n "$PROMETHEUS_ADDRESSES" ]]; then
  validate_addresses "PROMETHEUS_ADDRESSES"
else
  PROMETHEUS_ADDRESSES=$(generate_addresses "$KAFKA_PROMETHEUS_PORT")
fi
export PROMETHEUS_URLS=$(convert_to_prometheus_urls "$PROMETHEUS_ADDRESSES")

if [[ -n ${KAFKA_USER} && -n ${KAFKA_PASSWORD} ]]; then
  prepare_secured_config_files
else
  prepare_unsecured_config_files
fi
enrich_kafkactl_yml_with_ssl_configs

/sbin/tini -- /entrypoint.sh telegraf