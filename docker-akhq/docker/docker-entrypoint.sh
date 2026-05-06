#!/bin/bash

set -e

# Initialise writable runtime directories under /tmp so the container can run
# with readOnlyRootFilesystem: true.
AKHQ_WORK=/tmp/akhq
export MICRONAUT_CONFIG_FILES="${AKHQ_WORK}/application.yml"
mkdir -p "${AKHQ_WORK}/config/descs" "${AKHQ_WORK}/tls-ks"

# The section below decomposes the proto-files, that are added by operator according to the AKHQConfig CR
# The target directories DESCS_DEFAULT_DIR & DESCS_COMRESSED_DIR was already hardcoded & defined in operator functionality.

DESCS_DEFAULT_DIR="${AKHQ_WORK}/config/descs"
DESCS_COMRESSED_DIR="/app/config/descs-compressed"

if [[ -d "${DESCS_COMRESSED_DIR}" ]]; then
  if [[ -n "$(ls ${DESCS_COMRESSED_DIR})" ]]; then
      echo "Directory with compressed proto-files detected, start decompression..."
      if ! [[ -d "${DESCS_DEFAULT_DIR}" ]]; then
        mkdir -p "${DESCS_DEFAULT_DIR}"
      fi
      for file in ${DESCS_COMRESSED_DIR}/*; do
          filename=$(basename "$file")
          echo "Decompressing ${filename} ..."
          cp "$file" ${DESCS_DEFAULT_DIR}/${filename}
          gzip -d ${DESCS_DEFAULT_DIR}/${filename}
      done
      echo "Decompression finished"
  else
    echo "Directory with compressed proto-files detected, but it is empty. Skip decompression stage"
  fi
fi


#Resolve security protocol.
#
#$1 - security protocol without SSL
#$2 - security protocol with SSL
resolve_security_protocol() {
  local security_protocol="$1"
  if [[ "${KAFKA_ENABLE_SSL}" == "true" ]]; then
    security_protocol="$2"
  fi
  echo "$security_protocol"
}


# for local dev in docker compose provide the ability to specify configuration in native AKHQ way
if [[ -n "${AKHQ_CONFIGURATION}" ]]; then
  echo "AKHQ_CONFIGURATION is provided and ${AKHQ_WORK}/application.yml file will be overwritten with it"
  echo "${AKHQ_CONFIGURATION}" > ${AKHQ_WORK}/application.yml

else

  if [[ -z "${KAFKA_SERVICE_NAME}" ]]; then
    echo "KAFKA_SERVICE_NAME must be set"
    exit 1
  fi

  : ${KAFKA_SERVICE_NAME:="kafka"}
  : ${KAFKA_POLL_TIMEOUT_MS:=10000}
  : ${BOOTSTRAP_SERVERS:="${KAFKA_SERVICE_NAME}:9092"}
  : ${KAFKA_SASL_MECHANISM:="SCRAM-SHA-512"}
  echo "Kafka bootstrap servers: <${BOOTSTRAP_SERVERS}>, KAFKA_POLL_TIMEOUT_MS = ${KAFKA_POLL_TIMEOUT_MS}"

  if [[ -n ${KAFKA_AUTH_USERNAME} && -n ${KAFKA_AUTH_PASSWORD} ]]; then
    echo "Kafka credentials are provided, so SASL mechanism will be enabled"
    SECURITY_PROTOCOL=$(resolve_security_protocol "SASL_PLAINTEXT" "SASL_SSL")
    SASL_LOGIN_MODULE="org.apache.kafka.common.security.scram.ScramLoginModule"
    if [[ "${KAFKA_SASL_MECHANISM}" == "PLAIN" ]]; then
        SASL_LOGIN_MODULE="org.apache.kafka.common.security.plain.PlainLoginModule"
    fi
    SASL_CONFIGURATION="security.protocol: ${SECURITY_PROTOCOL}
        sasl.mechanism: ${KAFKA_SASL_MECHANISM}
        sasl.jaas.config: ${SASL_LOGIN_MODULE} required username=\"${KAFKA_AUTH_USERNAME}\" password=\"${KAFKA_AUTH_PASSWORD}\";"
  else
    SECURITY_PROTOCOL=$(resolve_security_protocol "PLAINTEXT" "SSL")
    SASL_CONFIGURATION="security.protocol: ${SECURITY_PROTOCOL}"
  fi

  if [[ "${KAFKA_ENABLE_SSL}" == "true" ]]; then
    kafka_ca_cert_path="/tls/ca.crt"
    kafka_tls_key_path="/tls/tls.key"
    kafka_tls_cert_path="/tls/tls.crt"
    kafka_tls_ks_dir="${AKHQ_WORK}/tls-ks"
    mkdir -p ${kafka_tls_ks_dir}
    if [[ -f $kafka_ca_cert_path ]]; then
      SSL_PASSWORD="changeit"
      if [[ -f $kafka_tls_key_path && -f $kafka_tls_cert_path ]]; then
        SSL_KEYSTORE_LOCATION="${kafka_tls_ks_dir}/kafka.keystore.jks"
        openssl pkcs12 -export -in $kafka_tls_cert_path -inkey $kafka_tls_key_path -out ${kafka_tls_ks_dir}/kafka.keystore.p12 -passout pass:${SSL_PASSWORD}
        keytool -importkeystore -destkeystore ${SSL_KEYSTORE_LOCATION} -deststorepass ${SSL_PASSWORD} \
          -srckeystore ${kafka_tls_ks_dir}/kafka.keystore.p12 -srcstoretype PKCS12 -srcstorepass ${SSL_PASSWORD}
        keytool -import -trustcacerts -keystore ${SSL_KEYSTORE_LOCATION} -storepass ${SSL_PASSWORD} -noprompt -alias ca-cert -file $kafka_ca_cert_path
        SSL_CONFIGURATION="ssl.keystore.location: ${SSL_KEYSTORE_LOCATION}
        ssl.keystore.password: ${SSL_PASSWORD}
        ssl.key.password: ${SSL_PASSWORD}"
      fi

      SSL_TRUSTSTORE_LOCATION="${kafka_tls_ks_dir}/kafka.truststore.jks"
      keytool -import -trustcacerts -keystore ${SSL_TRUSTSTORE_LOCATION} -storepass ${SSL_PASSWORD} -noprompt -alias ca -file $kafka_ca_cert_path

      SSL_CONFIGURATION="ssl.truststore.location: ${SSL_TRUSTSTORE_LOCATION}
        ssl.truststore.password: ${SSL_PASSWORD}
        ${SSL_CONFIGURATION}"
    fi
  fi

  CONNECT_CONFIGURATION=""
  if [[ -n ${STREAMING_PLATFORM_URL} ]]; then
    echo "Streaming Platform URL <${STREAMING_PLATFORM_URL}> is provided, so Kafka Connect will be configured"
    CONNECT_CREDENTIALS=""
    if [[ -n ${STREAMING_PLATFORM_USERNAME} && -n ${STREAMING_PLATFORM_PASSWORD} ]]; then
      CONNECT_CREDENTIALS="basic-auth-username: ${STREAMING_PLATFORM_USERNAME}
          basic-auth-password: ${STREAMING_PLATFORM_PASSWORD}"
    fi
    CONNECT_CONFIGURATION="connect:
        - name: streaming-platform
          url: ${STREAMING_PLATFORM_URL}
          ${CONNECT_CREDENTIALS}"
  fi

  # Configurations should have the right indentation, because YAML format is sensitive
  SHIFTED_SECURITY_GROUPS_CONFIGURATION=$(echo "${SECURITY_GROUPS_CONFIGURATION}"| sed 's/^/    /')
  SHIFTED_BASIC_AUTH_USERS_CONFIGURATION=$(echo "${BASIC_AUTH_USERS_CONFIGURATION}"| sed 's/^/    /')
  SHIFTED_LDAP_SERVER_CONFIGURATION=$(echo "${LDAP_SERVER_CONFIGURATION}"| sed 's/^/        /')
  SHIFTED_LDAP_USERS_CONFIGURATION=$(echo "${LDAP_USERS_CONFIGURATION}"| sed 's/^/    /')
  SHIFTED_PROTOBUF_CONFIGURATION=$(echo "${PROTOBUF_CONFIGURATION}"| sed 's/^/      /')

  if [[ -n ${LDAP_SERVER_CONFIGURATION} ]]; then
    echo "LDAP server configuration is provided, so LDAP connection will be configured for Micronaut"
    FULL_LDAP_SERVER_CONFIGURATION="ldap:
      default:
        enabled: true
${SHIFTED_LDAP_SERVER_CONFIGURATION}"
  fi

  DEFAULT_USER_CONFIGURATION=""
  if [[ -n ${AKHQ_DEFAULT_USER} && -n ${AKHQ_DEFAULT_PASSWORD} ]]; then
    echo "AKHQ default credentials are provided, so default user will be configured"
    sha256_password=$(echo -n ${AKHQ_DEFAULT_PASSWORD} | sha256sum)
    sha256_password=${sha256_password%% *}
    DEFAULT_USER_CONFIGURATION=\
"      - username: ${AKHQ_DEFAULT_USER}
        password: ${sha256_password}
        groups:
        - admin"

    # if BASIC_AUTH_USERS_CONFIGURATION is present it contains "basic-auth:",
    # otherwise DEFAULT_USER_CONFIGURATION should contain it
    if [[ -z ${BASIC_AUTH_USERS_CONFIGURATION} ]]; then
      DEFAULT_USER_CONFIGURATION=\
"    basic-auth:
${DEFAULT_USER_CONFIGURATION}"
    fi
  fi

  if [[ "${ENABLE_ACCESS_LOG}" == "true" ]]; then
    echo "Access log is enabled"
  else
    echo "Access log is disabled"
  fi

JWT_SECRET=$( tr -dc A-Za-z0-9 </dev/urandom | head -c 260 )

if [[ -n ${SCHEMA_REGISTRY_URL} ]]; then
  schema_registry_template="
        url: ${SCHEMA_REGISTRY_URL}
        type: ${SCHEMA_REGISTRY_TYPE}"

  if [[ -n ${SCHEMA_REGISTRY_USERNAME} && -n ${SCHEMA_REGISTRY_PASSWORD} ]]; then
    schema_registry_template="${schema_registry_template}
        basic-auth-username: ${SCHEMA_REGISTRY_USERNAME}
        basic-auth-password: ${SCHEMA_REGISTRY_PASSWORD}"
  fi

  if [[ $SCHEMA_REGISTRY_URL == *"https"* ]]; then
    schema_registry_template="${schema_registry_template}
        properties:
          ssl.protocol: TLS"
  fi

  SCHEMA_REGISTRY="schema-registry:${schema_registry_template}"
fi
MICRONAUT_SECURITY_CONFIG=""
if [[ -n ${DEFAULT_USER_CONFIGURATION} || -n ${BASIC_AUTH_USERS_CONFIGURATION} || -n ${LDAP_SERVER_CONFIGURATION} ]]; then
  MICRONAUT_SECURITY_CONFIG=\
"micronaut:
   security:
     enabled: true
     token:
       jwt:
         signatures:
           secret:
             generator:
               secret: ${JWT_SECRET}
     ${FULL_LDAP_SERVER_CONFIGURATION}"
fi

  cat > ${AKHQ_WORK}/application.yml << EOL
${MICRONAUT_SECURITY_CONFIG}
endpoints:
  all:
    port: 8081
akhq:
  server:
    access-log:
      enabled: ${ENABLE_ACCESS_LOG}
  connections:
    ${KAFKA_SERVICE_NAME}:
      properties:
        bootstrap.servers: ${BOOTSTRAP_SERVERS}
        ${SASL_CONFIGURATION}
        ${SSL_CONFIGURATION}
      ${SCHEMA_REGISTRY}
      ${CONNECT_CONFIGURATION}
${SHIFTED_PROTOBUF_CONFIGURATION}
  clients-defaults:
    consumer:
      properties:
        default.api.timeout.ms: ${DEFAULT_API_TIMEOUT_MS:=60000}
  ui-options:
    topic:
      default-view: ALL
  topic-data:
    poll-timeout: ${KAFKA_POLL_TIMEOUT_MS}
  security:
    default-group: no-roles
${SHIFTED_SECURITY_GROUPS_CONFIGURATION}
${SHIFTED_BASIC_AUTH_USERS_CONFIGURATION}
${DEFAULT_USER_CONFIGURATION}
${SHIFTED_LDAP_USERS_CONFIGURATION}
EOL

fi

echo "Import trustcerts to application keystore"
TRUST_CERTS_DIR=${AKHQ_HOME}/trustcerts
DESTINATION_KEYSTORE_PATH=${AKHQ_WORK}/cacerts
KEYSTORE_PATH=${JAVA_HOME}/lib/security/cacerts

echo "Copy Java cacerts to ${DESTINATION_KEYSTORE_PATH}"
${JAVA_HOME}/bin/keytool --importkeystore -noprompt \
        -srckeystore ${KEYSTORE_PATH} \
        -srcstorepass changeit \
        -destkeystore ${DESTINATION_KEYSTORE_PATH} \
        -deststorepass changeit &> /dev/null

if [[ "$(ls ${TRUST_CERTS_DIR})" ]]; then
    for filename in ${TRUST_CERTS_DIR}/*; do
        echo "Import $filename certificate to Java cacerts"
        ${JAVA_HOME}/bin/keytool -import -trustcacerts -keystore ${DESTINATION_KEYSTORE_PATH} -storepass changeit -noprompt -alias ${filename} -file ${filename}
    done;
fi

export JAVA_OPTS="$JAVA_OPTS $HEAP_OPTS -Djavax.net.ssl.trustStore=${DESTINATION_KEYSTORE_PATH} -Djavax.net.ssl.trustStorePassword=changeit"

exec "$@"
