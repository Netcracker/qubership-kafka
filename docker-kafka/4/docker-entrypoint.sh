#!/bin/bash

# Add missing EOF at the end of the config file
echo "" >> ${KAFKA_HOME}/config/server.properties

# Exit immediately if a *pipeline* returns a non-zero status. (Add -x for command tracing)
set -e
if [[ "$DEBUG" == true ]]; then
  set -x
  printenv
fi

if [[ ${DISABLE_SECURITY} == "false" && -z ${ADMIN_USERNAME} && -z ${ADMIN_PASSWORD} && -z ${CLIENT_USERNAME} && -z ${CLIENT_PASSWORD} ]]; then
  echo "Warning: All credential are not set, security is disabled"
  DISABLE_SECURITY=true
fi

if [[ -f ${KAFKA_HOME}/logs/errors.log ]]; then
  rm ${KAFKA_HOME}/logs/errors.log
fi

if [[ -z "$BROKER_ID" ]]; then
  BROKER_ID=1
  echo "WARNING: Using default BROKER_ID=1, which is valid only for non-clustered installations."
fi
if [[ -z "$REPLICATION_FACTOR" ]]; then
  REPLICATION_FACTOR=1
  echo "WARNING: Using default REPLICATION_FACTOR=1, which is valid only for non-clustered installations."
fi
if [[ -z "$ZOOKEEPER_CONNECT" ]]; then
  # Look for any environment variables set by Docker container linking. For example, if the container
  # running Zookeeper were named 'zoo' in this container, then Docker should have created several envs,
  # such as 'ZOO_PORT_2181_TCP'. If so, then use that to automatically set the 'zookeeper.connect' property.
  export ZOOKEEPER_CONNECT=$(env | grep .*PORT_2181_TCP= | sed -e 's|.*tcp://||' | uniq | paste -sd ,)
fi
if [[ "$KRAFT_ENABLED" != "true" ]]; then
  if [[ "x$ZOOKEEPER_CONNECT" = "x" ]]; then
    echo "The ZOOKEEPER_CONNECT variable must be set, or the container must be linked to one that runs Zookeeper."
    exit 1
  else
    echo "Using ZOOKEEPER_CONNECT=$ZOOKEEPER_CONNECT"
  fi
fi
if [[ -n "$HEAP_OPTS" ]]; then
  export KAFKA_HEAP_OPTS=${HEAP_OPTS}
  unset HEAP_OPTS
fi

: ${BROKER_HOST_NAME:=${HOSTNAME}}
: ${EXTENDED_JMX_CONFIG:=false}
jmx_config="${KAFKA_HOME}/config/jmx-exporter-config.yml"
if [[ "$EXTENDED_JMX_CONFIG" == true ]]; then
  jmx_config="${KAFKA_HOME}/config/extended-jmx-exporter-config.yml"
fi
cp -f ${jmx_config} ${KAFKA_HOME}/config/jmx-exporter.yml
sed -i -e "s/\${BROKER_HOST_NAME}/$BROKER_HOST_NAME/g" ${KAFKA_HOME}/config/jmx-exporter.yml

export KAFKA_OPTS="${KAFKA_OPTS} -javaagent:/opt/kafka/libs/jmx_prometheus_javaagent-1.1.0.jar=8080:${KAFKA_HOME}/config/jmx-exporter.yml"

if [[ "$KRAFT_ENABLED" == "true" ]]; then
  export CONF_KAFKA_PROCESS_ROLES=${PROCESS_ROLES}
  export CONF_KAFKA_CONTROLLER_LISTENER_NAMES=CONTROLLER
  export CONF_KAFKA_CONTROLLER_QUORUM_VOTERS=${VOTERS}
  export CONF_KAFKA_NODE_ID=${BROKER_ID}
  # For Kraft remove quorum-state file so that we won't enter voter not match error after scaling up/down https://issues.apache.org/jira/browse/KAFKA-14094
  if [ -f "/var/opt/kafka/data/$BROKER_ID/__cluster_metadata-0/quorum-state" ]; then
    echo "Removing quorum-state file"
    rm -f "/var/opt/kafka/data/$BROKER_ID/__cluster_metadata-0/quorum-state"
  fi
fi

### see: https://cwiki.apache.org/confluence/display/KAFKA/KIP-537%3A+Increase+default+zookeeper+session+timeout
export CONF_KAFKA_ZOOKEEPER_CONNECTION_TIMEOUT_MS=${CONF_KAFKA_ZOOKEEPER_CONNECTION_TIMEOUT_MS:=18000}
export CONF_KAFKA_ZOOKEEPER_SESSION_TIMEOUT_MS=${CONF_KAFKA_ZOOKEEPER_SESSION_TIMEOUT_MS:=18000}
export CONF_KAFKA_REPLICA_LAG_TIME_MAX_MS=${CONF_KAFKA_REPLICA_LAG_TIME_MAX_MS:=30000}
###
export CONF_KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS=${CONF_KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS:=3000}
export CONF_KAFKA_DELETE_TOPIC_ENABLE=true
export CONF_KAFKA_ZOOKEEPER_CONNECT=${ZOOKEEPER_CONNECT}
export CONF_KAFKA_BROKER_ID=${BROKER_ID}
export CONF_KAFKA_LOG_DIRS="$KAFKA_DATA_DIRS/$BROKER_ID"

if [[ -z "$CONF_KAFKA_DEFAULT_REPLICATION_FACTOR" ]]; then
  # REPLICATION_FACTOR = 3 is a recommended setting for a cluster of more than 3 brokers
  if [[ "$REPLICATION_FACTOR" -gt "3" ]]; then
    REPLICATION_FACTOR=3
  fi
else
  # if replication factor is set via CONF_KAFKA_DEFAULT_REPLICATION_FACTOR variable, it will be used
  REPLICATION_FACTOR=${CONF_KAFKA_DEFAULT_REPLICATION_FACTOR}
fi

export CONF_KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=${REPLICATION_FACTOR}
export CONF_KAFKA_DEFAULT_REPLICATION_FACTOR=${REPLICATION_FACTOR}
export CONF_KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR=${REPLICATION_FACTOR}
mkdir -p ${CONF_KAFKA_LOG_DIRS}

MIN_ISR=$(( ($REPLICATION_FACTOR / 2) + 1 ))
# Kafka doesn't have arbiter node, so min insync replicas should be specified to 1
if [[ "$REPLICATION_FACTOR" -eq 2 ]]; then
  MIN_ISR=1
fi
export CONF_KAFKA_TRANSACTION_STATE_LOG_MIN_ISR=${CONF_KAFKA_TRANSACTION_STATE_LOG_MIN_ISR:=${MIN_ISR}}
export CONF_KAFKA_MIN_INSYNC_REPLICAS=${CONF_KAFKA_MIN_INSYNC_REPLICAS:=${MIN_ISR}}

unset BROKER_ID
unset REPLICATION_FACTOR
unset ZOOKEEPER_CONNECT

if [[ -z "$HOST_NAME" ]]; then
  HOST_NAME=$(ip addr | grep 'BROADCAST' -A2 | tail -n1 | awk '{print $2}' | cut -f1  -d'/')
fi

# Resolve security protocol.
#
#$1 - security protocol without SSL
#$2 - security protocol with SSL
resolve_security_protocol() {
  local security_protocol="$1"
  if [[ "${ENABLE_SSL}" == "true" ]]; then
    security_protocol="$2"
  fi
  echo "$security_protocol"
}

if [[ "$DISABLE_SECURITY" == false ]]; then
  SECURITY_PROTOCOL=$(resolve_security_protocol "SASL_PLAINTEXT" "SASL_SSL")
  NONENCRYPTED_SECURITY_PROTOCOL=SASL_PLAINTEXT
else
  SECURITY_PROTOCOL=$(resolve_security_protocol "PLAINTEXT" "SSL")
  NONENCRYPTED_SECURITY_PROTOCOL=PLAINTEXT
fi

: ${INTERNAL_PORT:=${CLIENT_PORT:-9092}}
: ${INTER_BROKER_PORT:=9093}
: ${LISTENER_EXTERNAL_PORT:=9094}
: ${EXTERNAL_PORT:=${LISTENER_EXTERNAL_PORT}}
: ${NONENCRYPTED_PORT:=9095}
: ${INTERNAL_HOST_NAME:=${CLIENT_HOST_NAME:-${HOST_NAME}}}
: ${INTER_BROKER_HOST_NAME:=${HOST_NAME}}
: ${READINESS_PERIOD:=30}

if [[ -n "$EXTERNAL_HOST_NAME" ]]; then
  ENABLE_EXTERNAL_LISTENER=true
fi

if [[ -z "$LISTENERS" ]]; then
  LISTENERS=INTERNAL://0.0.0.0:${INTERNAL_PORT},INTER_BROKER://0.0.0.0:${INTER_BROKER_PORT}
  if [[ "$KRAFT_ENABLED" == "true" && "$MIGRATED_BROKER" != "true" ]]; then
    LISTENERS=${LISTENERS},CONTROLLER://0.0.0.0:9096
  fi
  if [[ "$ENABLE_EXTERNAL_LISTENER" == true ]]; then
    LISTENERS=${LISTENERS},EXTERNAL://0.0.0.0:${LISTENER_EXTERNAL_PORT}
  fi
  if [[ "${ENABLE_SSL}" == "true" && "${ALLOW_NONENCRYPTED_ACCESS}" == "true" ]]; then
    LISTENERS=${LISTENERS},NONENCRYPTED://0.0.0.0:${NONENCRYPTED_PORT}
  fi
fi
if [[ -z "$ADVERTISED_LISTENERS" && "$MIGRATION_CONTROLLER" != "true" && "$MIGRATED_CONTROLLER" != "true" ]]; then
  ADVERTISED_LISTENERS=INTERNAL://${INTERNAL_HOST_NAME}:${INTERNAL_PORT},INTER_BROKER://${INTER_BROKER_HOST_NAME}:${INTER_BROKER_PORT}
  if [[ "$ENABLE_EXTERNAL_LISTENER" == true ]]; then
    ADVERTISED_LISTENERS=${ADVERTISED_LISTENERS},EXTERNAL://${EXTERNAL_HOST_NAME}:${EXTERNAL_PORT}
  fi
  if [[ "${ENABLE_SSL}" == "true" && "${ALLOW_NONENCRYPTED_ACCESS}" == "true" ]]; then
    ADVERTISED_LISTENERS=${ADVERTISED_LISTENERS},NONENCRYPTED://${INTERNAL_HOST_NAME}:${NONENCRYPTED_PORT}
  fi
fi
if [[ -z "$LISTENER_SECURITY_PROTOCOL_MAP" ]]; then
  LISTENER_SECURITY_PROTOCOL_MAP=INTERNAL:${SECURITY_PROTOCOL},INTER_BROKER:${SECURITY_PROTOCOL}
  if [[ "$KRAFT_ENABLED" == "true" ]]; then
    LISTENER_SECURITY_PROTOCOL_MAP=${LISTENER_SECURITY_PROTOCOL_MAP},CONTROLLER:${SECURITY_PROTOCOL}
  fi
  if [[ "$ENABLE_EXTERNAL_LISTENER" == true ]]; then
    LISTENER_SECURITY_PROTOCOL_MAP=${LISTENER_SECURITY_PROTOCOL_MAP},EXTERNAL:${SECURITY_PROTOCOL}
  fi
  if [[ "${ENABLE_SSL}" == "true" && "${ALLOW_NONENCRYPTED_ACCESS}" == "true" ]]; then
    LISTENER_SECURITY_PROTOCOL_MAP=${LISTENER_SECURITY_PROTOCOL_MAP},NONENCRYPTED:${NONENCRYPTED_SECURITY_PROTOCOL}
  fi
fi
: ${INTER_BROKER_LISTENER_NAME:=INTER_BROKER}


if [[ "$MIGRATION_CONTROLLER" == "true" && "$MIGRATED_CONTROLLER" != "true" ]]; then
  export CONF_KAFKA_ZOOKEEPER_METADATA_MIGRATION_ENABLE="true"
fi

if [[ "$MIGRATION_BROKER" == "true" ]]; then
  export CONF_KAFKA_ZOOKEEPER_METADATA_MIGRATION_ENABLE="true"
  LISTENER_SECURITY_PROTOCOL_MAP=${LISTENER_SECURITY_PROTOCOL_MAP},CONTROLLER:${SECURITY_PROTOCOL}
  export CONF_KAFKA_CONTROLLER_QUORUM_VOTERS=${VOTERS}
  export CONF_KAFKA_CONTROLLER_LISTENER_NAMES=CONTROLLER
fi

export CONF_KAFKA_LISTENERS=${LISTENERS}
if [[ "$MIGRATION_CONTROLLER" != "true" && "$MIGRATED_CONTROLLER" != "true" ]]; then
  export CONF_KAFKA_ADVERTISED_LISTENERS=${ADVERTISED_LISTENERS}
fi
export CONF_KAFKA_LISTENER_SECURITY_PROTOCOL_MAP=${LISTENER_SECURITY_PROTOCOL_MAP}
export CONF_KAFKA_INTER_BROKER_LISTENER_NAME=${INTER_BROKER_LISTENER_NAME}

# Parse client listeners names and store them in array
function parse_client_listeners() {
  clients_listeners_names=()
  local pairs=$(echo ${LISTENER_SECURITY_PROTOCOL_MAP} | tr "," "\n")
  for pair in ${pairs}; do
    local listener_name=$(echo ${pair} | cut -d ':' -f 1)
    if [[ ${listener_name} != "INTER_BROKER" && ${listener_name} != "CONTROLLER" ]]; then
      clients_listeners_names+=(${listener_name})
    fi
  done
}
parse_client_listeners

unset HOST_NAME
unset INTERNAL_PORT
unset INTER_BROKER_PORT
unset LISTENER_EXTERNAL_PORT
unset EXTERNAL_PORT
unset INTERNAL_HOST_NAME
unset INTER_BROKER_HOST_NAME
unset EXTERNAL_HOST_NAME
unset LISTENERS
unset ADVERTISED_LISTENERS
unset LISTENER_SECURITY_PROTOCOL_MAP
unset INTER_BROKER_LISTENER_NAME
echo "Using CONF_KAFKA_LISTENERS=$CONF_KAFKA_LISTENERS"
echo "Using CONF_KAFKA_ADVERTISED_LISTENERS=$CONF_KAFKA_ADVERTISED_LISTENERS"
echo "Using CONF_KAFKA_LISTENER_SECURITY_PROTOCOL_MAP=$CONF_KAFKA_LISTENER_SECURITY_PROTOCOL_MAP"
echo "Using CONF_KAFKA_INTER_BROKER_LISTENER_NAME=$CONF_KAFKA_INTER_BROKER_LISTENER_NAME"

#
# Set up the JMX options
#
: ${JMXAUTH:="false"}
: ${JMXSSL:="false"}
if [[ -n "$JMXPORT" && -n "$JMXHOST" ]]; then
  echo "Enabling JMX on ${JMXHOST}:${JMXPORT}"
  export KAFKA_JMX_OPTS="-Djava.rmi.server.hostname=${JMXHOST} -Dcom.sun.management.jmxremote.rmi.port=${JMXPORT} -Dcom.sun.management.jmxremote.port=${JMXPORT} -Dcom.sun.management.jmxremote -Dcom.sun.management.jmxremote.authenticate=${JMXAUTH} -Dcom.sun.management.jmxremote.ssl=${JMXSSL} "
fi

mkdir -p "${KAFKA_DATA_DIRS}/dumps"
export KAFKA_OPTS="-XX:+HeapDumpOnOutOfMemoryError -XX:HeapDumpPath=${KAFKA_DATA_DIRS}/dumps/ -XX:+ExitOnOutOfMemoryError ${KAFKA_OPTS} "

echo "Import trustcerts to application keystore"

TRUST_CERTS_DIR=${KAFKA_HOME}/trustcerts
DESTINATION_TRUSTSTORE_PATH=${KAFKA_HOME}/config/cacerts

KEYSTORE_PATH=${JAVA_HOME}/lib/security/cacerts

echo "Copy Java cacerts to $DESTINATION_TRUSTSTORE_PATH"
${JAVA_HOME}/bin/keytool --importkeystore -noprompt \
        -srckeystore ${KEYSTORE_PATH} \
        -srcstorepass changeit \
        -destkeystore ${DESTINATION_TRUSTSTORE_PATH} \
        -deststorepass changeit &> /dev/null

if [[ "$(ls ${TRUST_CERTS_DIR})" ]]; then
    for filename in ${TRUST_CERTS_DIR}/*; do
        echo "Import $filename certificate to Java cacerts"
        ${JAVA_HOME}/bin/keytool -import -trustcacerts -keystore ${DESTINATION_TRUSTSTORE_PATH} -storepass changeit -noprompt -alias ${filename} -file ${filename}
    done;
fi

export KAFKA_OPTS="${KAFKA_OPTS} -Djavax.net.ssl.trustStore=${KAFKA_HOME}/config/cacerts -Djavax.net.ssl.trustStorePassword=changeit"

PUBLIC_CERTS_DIR=${KAFKA_HOME}/public-certs
JWK_KEYSTORE_PATH=${KAFKA_HOME}/config/public_certs.jks

if [[ "$(ls ${PUBLIC_CERTS_DIR})" ]]; then
    for filename in ${PUBLIC_CERTS_DIR}/*; do
        local_filename="${filename##*/}"
        kid="${local_filename%.*}"
        echo "Import ${filename} certificate with public key id ${kid} to ${JWK_KEYSTORE_PATH} keystore"
        ${JAVA_HOME}/bin/keytool -import -trustcacerts -keystore ${JWK_KEYSTORE_PATH} -storepass changeit -noprompt -alias ${kid} -file ${filename}
    done;
fi

if [[ "${ENABLE_SSL}" == "true" ]]; then
  echo "Configuring Kafka TLS..."
  kafka_tls_dir=${KAFKA_HOME}/tls
  SSL_KEY_LOCATION=${kafka_tls_dir}/tls.key
  SSL_CERTIFICATE_LOCATION=${kafka_tls_dir}/tls.crt
  SSL_CA_LOCATION=${kafka_tls_dir}/ca.crt
  kafka_tls_ks_dir=${KAFKA_HOME}/tls-ks
  SSL_KEYSTORE_LOCATION=${kafka_tls_ks_dir}/kafka.keystore.jks
  SSL_TRUSTSTORE_LOCATION=${kafka_tls_ks_dir}/kafka.truststore.jks

  if [[ -f ${SSL_KEY_LOCATION} && -f ${SSL_CERTIFICATE_LOCATION} && -f ${SSL_CA_LOCATION} ]]; then
    mkdir -p ${kafka_tls_ks_dir}
    openssl pkcs12 -export -in ${SSL_CERTIFICATE_LOCATION} -inkey ${SSL_KEY_LOCATION} -out ${kafka_tls_ks_dir}/kafka.keystore.p12 -passout pass:changeit
    keytool -importkeystore -destkeystore ${SSL_KEYSTORE_LOCATION} -deststorepass changeit -srcstoretype PKCS12 -srckeystore ${kafka_tls_ks_dir}/kafka.keystore.p12 -srcstorepass changeit
    keytool -import -trustcacerts -keystore ${SSL_KEYSTORE_LOCATION} -storepass changeit -noprompt -alias ca-cert -file ${SSL_CA_LOCATION}
    keytool -import -trustcacerts -keystore ${SSL_TRUSTSTORE_LOCATION} -storepass changeit -noprompt -alias ca -file ${SSL_CA_LOCATION}

    export CONF_KAFKA_SSL_KEYSTORE_LOCATION=${SSL_KEYSTORE_LOCATION}
    export CONF_KAFKA_SSL_KEYSTORE_PASSWORD=changeit
    export CONF_KAFKA_SSL_KEY_PASSWORD=changeit
    export CONF_KAFKA_SSL_TRUSTSTORE_LOCATION=${SSL_TRUSTSTORE_LOCATION}
    export CONF_KAFKA_SSL_TRUSTSTORE_PASSWORD=changeit
    export CONF_KAFKA_SSL_CIPHER_SUITES=${SSL_CIPHER_SUITES}
    echo "Kafka TLS configuration is applied"
  else
    echo "Kafka TLS certificates must be provided."
    exit 1
  fi
fi

if [[ "${ENABLE_ZOOKEEPER_SSL}" == "true" ]]; then
  echo "Configuring ZooKeeper TLS..."
  cat > ${KAFKA_HOME}/bin/zk-tls-config.properties << EOL
zookeeper.ssl.client.enable=true
zookeeper.clientCnxnSocket=org.apache.zookeeper.ClientCnxnSocketNetty
EOL
  export CONF_KAFKA_ZOOKEEPER_SSL_CLIENT_ENABLE=true
  echo "zookeeper.clientCnxnSocket=org.apache.zookeeper.ClientCnxnSocketNetty" >> ${KAFKA_HOME}/config/server.properties

  zookeeper_tls_dir=${KAFKA_HOME}/zookeeper-tls
  zookeeper_ca_cert_path=${zookeeper_tls_dir}/ca.crt
  zookeeper_tls_key_path=${zookeeper_tls_dir}/tls.key
  zookeeper_tls_cert_path=${zookeeper_tls_dir}/tls.crt

  keystore_path=
  truststore_path=
  if [[ -f ${zookeeper_ca_cert_path} ]]; then
    zookeeper_tls_ks_dir=${KAFKA_HOME}/zookeeper-tls-ks
    mkdir -p ${zookeeper_tls_ks_dir}
    if [[ -f $zookeeper_tls_key_path && -f $zookeeper_tls_cert_path ]]; then
      keystore_path=${zookeeper_tls_ks_dir}/zookeeper.keystore.jks
      openssl pkcs12 -export -in ${zookeeper_tls_cert_path} -inkey ${zookeeper_tls_key_path} -out ${zookeeper_tls_ks_dir}/zookeeper.keystore.p12 -passout pass:changeit
      keytool -importkeystore -destkeystore ${keystore_path} -deststorepass changeit -srcstoretype PKCS12 -srckeystore ${zookeeper_tls_ks_dir}/zookeeper.keystore.p12 -srcstorepass changeit
      keytool -import -trustcacerts -keystore ${keystore_path} -storepass changeit -noprompt -alias ca-cert -file ${zookeeper_ca_cert_path}
    fi
    truststore_path=${zookeeper_tls_ks_dir}/zookeeper.truststore.jks
    keytool -import -trustcacerts -keystore ${truststore_path} -storepass changeit -noprompt -alias ca -file ${zookeeper_ca_cert_path}
  elif [[ "${ENABLE_SSL}" == "true" ]]; then
    keystore_path=${SSL_KEYSTORE_LOCATION}
    truststore_path=${SSL_TRUSTSTORE_LOCATION}
  fi
#  if [[ -n "${keystore_path}" ]]; then
#    export CONF_KAFKA_ZOOKEEPER_SSL_KEYSTORE_LOCATION=${keystore_path}
#    export CONF_KAFKA_ZOOKEEPER_SSL_KEYSTORE_PASSWORD=changeit
#    cat >> ${KAFKA_HOME}/bin/zk-tls-config.properties << EOL
#zookeeper.ssl.keystore.location=${keystore_path}
#zookeeper.ssl.keystore.password=changeit
#EOL
#    echo "ZooKeeper mTLS configuration is applied"
#  fi
  # Always configure ZooKeeper truststore.
  # - If truststore jks is specified, it will be used.
  # - If ca.crt file exists, truststore jks will be imported.
  # - Otherwise, the default truststore with imported certificates is used.
  if [[ -z "${truststore_path}" ]]; then
    truststore_path=${DESTINATION_TRUSTSTORE_PATH}
  fi
  export CONF_KAFKA_ZOOKEEPER_SSL_TRUSTSTORE_LOCATION=${truststore_path}
  export CONF_KAFKA_ZOOKEEPER_SSL_TRUSTSTORE_PASSWORD=changeit
  cat >> ${KAFKA_HOME}/bin/zk-tls-config.properties << EOL
zookeeper.ssl.truststore.location=${truststore_path}
zookeeper.ssl.truststore.password=changeit
EOL
  echo "ZooKeeper TLS configuration is applied"
else
  cat > ${KAFKA_HOME}/bin/zk-tls-config.properties << EOL
zookeeper.ssl.client.enable=false
EOL
  export CONF_KAFKA_ZOOKEEPER_SSL_CLIENT_ENABLE=false
fi

# Creates or updates user credentials in Zookeeper
# $1 - username
# $2 - password
function create_user() {
  echo "Create new user with name $1"
  # https://cwiki.apache.org/confluence/display/KAFKA/KIP-554%3A+Add+Broker-side+SCRAM+Config+API
  # Move the logic to Kafka Operator after Kafka upgrading to v.2.7.0 to remove dependency on ZooKeeper
  ${KAFKA_HOME}/bin/kafka-configs.sh \
    --zookeeper "$CONF_KAFKA_ZOOKEEPER_CONNECT" \
    --zk-tls-config-file ${KAFKA_HOME}/bin/zk-tls-config.properties \
    --alter \
    --add-config "SCRAM-SHA-512=[password=${2}]" \
    --entity-type users \
    --entity-name ${1}
}

# Configures sasl mechanisms on listener.
#
#$1 - listener name lowercase
function configure_sasl_mechanisms_on_listener() {
  local listener_name="$1"
  echo "Configure sasl mechanisms on $listener_name listener"
  echo "listener.name.$listener_name.oauthbearer.sasl.jaas.config=org.apache.kafka.common.security.oauthbearer.OAuthBearerLoginModule required \
  clockSkew=\"$CLOCK_SKEW\" \
  jwksConnectionTimeout=\"$JWKS_CONNECTION_TIMEOUT\" \
  jwksReadTimeout=\"$JWKS_READ_TIMEOUT\" \
  jwksSizeLimit=\"$JWKS_SIZE_LIMIT\" \
  idpWhitelist=\"$IDP_WHITELIST\" \
  jwkSourceType=\"$JWK_SOURCE_TYPE\" \
  keystorePath=\"${JWK_KEYSTORE_PATH}\" \
  keystorePassword=\"changeit\" \
  auditLogsEnabled=\"${ENABLE_AUDIT_LOGS}\" \
  auditCefConfigPath=\"${KAFKA_HOME}/config/cef-configuration.xml\" \
  tokenRolesPath=\"$TOKEN_ROLES_PATH\";" \
    >> ${KAFKA_HOME}/config/server.properties
  echo "listener.name.$listener_name.scram-sha-512.sasl.jaas.config=org.apache.kafka.common.security.scram.ScramLoginModule required \
  username=\"${CLIENT_USERNAME}\" \
  password=\"${CLIENT_PASSWORD}\";" \
    >> ${KAFKA_HOME}/config/server.properties
  if [[ "${ENABLE_SSL}" == "true" ]]; then
    if [[ ${listener_name} != "nonencrypted" ]]; then
      echo "listener.name.$listener_name.plain.sasl.jaas.config=org.apache.kafka.common.security.plain.PlainLoginModule required \
  \"user_${CLIENT_USERNAME}\"=\"${CLIENT_PASSWORD}\";" \
        >> ${KAFKA_HOME}/config/server.properties
    else
      echo "listener.name.$listener_name.sasl.enabled.mechanisms=SCRAM-SHA-512,OAUTHBEARER" \
        >> ${KAFKA_HOME}/config/server.properties
    fi
  fi
}

#Set OAuth and SASL, SSL parameters on listener
#
#$1 - listener name lowercase
function set_security_parameters() {
  local listener_name="$1"
  echo "Adding security for $1 listener"
  configure_sasl_mechanisms_on_listener "${listener_name}"

  env_name=CONF_KAFKA_LISTENER_NAME_${listener_name}_OAUTHBEARER_SASL_LOGIN_CALLBACK_HANDLER_CLASS
  export ${env_name}=org.qubership.kafka.security.oauthbearer.OAuthBearerLoginCallbackHandler
  echo "Using ${env_name}=${!env_name}"

  env_name=CONF_KAFKA_LISTENER_NAME_${listener_name}_OAUTHBEARER_SASL_SERVER_CALLBACK_HANDLER_CLASS
  export ${env_name}=org.qubership.kafka.security.oauthbearer.OAuthBearerValidatorCallbackHandler
  echo "Using ${env_name}=${!env_name}"

  env_name=CONF_KAFKA_LISTENER_NAME_${listener_name}_OAUTHBEARER_CONNECTIONS_MAX_REAUTH_MS
  export ${env_name}=${!env_name:=3600000}
  echo "Using ${env_name}=${!env_name}"

  if [[ "${ENABLE_SSL}" == "true" && "${ENABLE_2WAY_SSL}" == "true" && ${listener_name} != "nonencrypted" ]]; then
    env_name=CONF_KAFKA_LISTENER_NAME_${listener_name}_SSL_CLIENT_AUTH
    export ${env_name}=required
    echo "Using ${env_name}=${!env_name}"
  fi
}

if [[ "$KRAFT_ENABLED" != "true" || "$MIGRATION_CONTROLLER" == "true" ]]; then
  if [[ -n ${ZOOKEEPER_CLIENT_USERNAME} && -n ${ZOOKEEPER_CLIENT_PASSWORD} ]]; then
    if [[ ${ZOOKEEPER_SET_ACL}  == "true" ]]; then
      export CONF_KAFKA_ZOOKEEPER_SET_ACL=true
    fi
    cat >> ${KAFKA_HOME}/config/kafka_jaas.conf << EOL
Client {
       org.apache.zookeeper.server.auth.DigestLoginModule required
       username="${ZOOKEEPER_CLIENT_USERNAME}"
       password="${ZOOKEEPER_CLIENT_PASSWORD}";
};
EOL
    if [[ ${KAFKA_OPTS} != *"-Djava.security.auth.login.config"* ]]; then
      export KAFKA_OPTS="${KAFKA_OPTS} -Djava.security.auth.login.config=${KAFKA_HOME}/config/kafka_jaas.conf"
    fi
  else
    export KAFKA_OPTS="${KAFKA_OPTS} -Dzookeeper.sasl.client=false"
  fi
fi

#
# Setup authentication
#
if [[ "$DISABLE_SECURITY" == false ]]; then
  echo "Configure security"
  if [[ "$KRAFT_ENABLED" == "true" || "$MIGRATION_BROKER" == "true" ]]; then
    KAFKA_CREDENTIALS="--add-scram SCRAM-SHA-512=[name=${ADMIN_USERNAME},password=${ADMIN_PASSWORD}] --add-scram SCRAM-SHA-512=[name=${CLIENT_USERNAME},password=${CLIENT_PASSWORD}]"
  else
    create_user ${ADMIN_USERNAME} ${ADMIN_PASSWORD}
    create_user ${CLIENT_USERNAME} ${CLIENT_PASSWORD}
  fi
  echo "Create jaas config file"
  cat >> ${KAFKA_HOME}/config/kafka_jaas.conf << EOL
inter_broker.KafkaServer {
    org.apache.kafka.common.security.scram.ScramLoginModule required
    username="${ADMIN_USERNAME}"
    password="${ADMIN_PASSWORD}";
};

EOL

  if [[ "$KRAFT_ENABLED" == "true" || "$MIGRATION_BROKER" == "true" ]]; then
    cat >> ${KAFKA_HOME}/config/kafka_jaas.conf << EOL
internal.KafkaServer {
    org.apache.kafka.common.security.scram.ScramLoginModule required
    username="${ADMIN_USERNAME}"
    password="${ADMIN_PASSWORD}";
};

EOL

    cat >> ${KAFKA_HOME}/config/kafka_jaas.conf << EOL
controller.KafkaServer {
    org.apache.kafka.common.security.plain.PlainLoginModule required
    username="${ADMIN_USERNAME}"
    password="${ADMIN_PASSWORD}"
    "user_${ADMIN_USERNAME}"="${ADMIN_PASSWORD}";
};

EOL
  fi

  export KAFKA_OPTS="${KAFKA_OPTS} -Djava.security.auth.login.config=${KAFKA_HOME}/config/kafka_jaas.conf"

  export CONF_KAFKA_SASL_MECHANISM_INTER_BROKER_PROTOCOL=SCRAM-SHA-512
  echo "Using CONF_KAFKA_SASL_MECHANISM_INTER_BROKER_PROTOCOL=$CONF_KAFKA_SASL_MECHANISM_INTER_BROKER_PROTOCOL"
  if [[ "$KRAFT_ENABLED" == "true" || "$MIGRATION_BROKER" == "true" ]]; then
    export CONF_KAFKA_SASL_MECHANISM_CONTROLLER_PROTOCOL=PLAIN
    echo "Using CONF_KAFKA_SASL_MECHANISM_CONTROLLER_PROTOCOL=$CONF_KAFKA_SASL_MECHANISM_CONTROLLER_PROTOCOL"
    SASL_ENABLED_MECHANISMS="PLAIN,SCRAM-SHA-512,OAUTHBEARER"
  else
    SASL_ENABLED_MECHANISMS="SCRAM-SHA-512,OAUTHBEARER"
  fi
  if [[ "${ENABLE_SSL}" == "true" ]]; then
    SASL_ENABLED_MECHANISMS="PLAIN,${SASL_ENABLED_MECHANISMS}"
  fi
  export CONF_KAFKA_SASL_ENABLED_MECHANISMS=${SASL_ENABLED_MECHANISMS}
  echo "Using CONF_KAFKA_SASL_ENABLED_MECHANISMS=$CONF_KAFKA_SASL_ENABLED_MECHANISMS"

  for (( i=0; i<${#clients_listeners_names[@]}; i++ )); do
    set_security_parameters $(echo "${clients_listeners_names[i]}" | tr '[:upper:]' '[:lower:]')
  done

  export CONF_KAFKA_PRINCIPAL_BUILDER_CLASS=org.qubership.kafka.security.authorization.ExtendedKafkaPrincipalBuilder
  echo "Using CONF_KAFKA_PRINCIPAL_BUILDER_CLASS=$CONF_KAFKA_PRINCIPAL_BUILDER_CLASS"

  if [[ "$ENABLE_AUTHORIZATION" == true ]]; then
    if [[ "$KRAFT_ENABLED" != "true" ]]; then
      export CONF_KAFKA_AUTHORIZER_CLASS_NAME=org.qubership.kafka.security.authorization.ExtendedAclAuthorizer
    else
      export CONF_KAFKA_AUTHORIZER_CLASS_NAME=org.qubership.kafka.security.authorization.ExtendedStandardAuthorizer
    fi
    echo "Using CONF_KAFKA_AUTHORIZER_CLASS_NAME=$CONF_KAFKA_AUTHORIZER_CLASS_NAME"
    export CONF_KAFKA_SUPER_USERS="User:$ADMIN_USERNAME;User:$CLIENT_USERNAME"
    echo "Using CONF_KAFKA_SUPER_USERS=$CONF_KAFKA_SUPER_USERS"
  else
    echo "WARNING! Authorization is disabled, its configuration is skipped!"
  fi
else
  echo "WARNING! Security is disabled, authentication configuration skipped!"
fi

function enrich_adminclient_properties_with_ssl_configs() {
  if [[ "${ENABLE_SSL}" == "true" ]]; then
    cat >> ${KAFKA_HOME}/bin/adminclient.properties << EOL
ssl.truststore.location=${SSL_TRUSTSTORE_LOCATION}
ssl.truststore.password=changeit
EOL
    if [[ "${ENABLE_2WAY_SSL}" == "true" ]]; then
      cat >> ${KAFKA_HOME}/bin/adminclient.properties << EOL
ssl.keystore.location=${SSL_KEYSTORE_LOCATION}
ssl.keystore.password=changeit
ssl.key.password=changeit
EOL
    fi
  fi
}

function enrich_kcat_properties_with_ssl_configs() {
  if [[ "${ENABLE_SSL}" == "true" ]]; then
    cat >> ${KAFKA_HOME}/bin/kcat.properties << EOL
ssl.ca.location=${SSL_CA_LOCATION}
EOL
    if [[ "${ENABLE_2WAY_SSL}" == "true" ]]; then
      cat >> ${KAFKA_HOME}/bin/kcat.properties << EOL
ssl.key.location=${SSL_KEY_LOCATION}
ssl.certificate.location=${SSL_CERTIFICATE_LOCATION}
EOL
    fi
  fi
}

function enrich_kafkactl_yml_with_ssl_configs() {
  if [[ "${ENABLE_SSL}" == "true" ]]; then
    cat >> ${KAFKA_HOME}/bin/kafkactl.yml << EOL
    tls:
      enabled: true
      ca: ${SSL_CA_LOCATION}
EOL
    if [[ "${ENABLE_2WAY_SSL}" == "true" ]]; then
      cat >> ${KAFKA_HOME}/bin/kafkactl.yml << EOL
      certKey: ${SSL_KEY_LOCATION}
      cert: ${SSL_CERTIFICATE_LOCATION}
EOL
    fi
  fi
}

function prepare_secured_config_files() {
  cat > ${KAFKA_HOME}/bin/adminclient.properties << EOL
security.protocol=${SECURITY_PROTOCOL}
sasl.mechanism=SCRAM-SHA-512
sasl.jaas.config=org.apache.kafka.common.security.scram.ScramLoginModule required username="${CLIENT_USERNAME}" password="${CLIENT_PASSWORD}";
EOL
  enrich_adminclient_properties_with_ssl_configs
  cat > ${KAFKA_HOME}/bin/kcat.properties << EOL
request.timeout.ms=$((${READINESS_PERIOD}*1000))
security.protocol=${SECURITY_PROTOCOL}
sasl.mechanisms=SCRAM-SHA-512
sasl.username=${CLIENT_USERNAME}
sasl.password=${CLIENT_PASSWORD}
EOL
  enrich_kcat_properties_with_ssl_configs
  cat > ${KAFKA_HOME}/bin/kafkactl.yml << EOL
current-context: default
contexts:
  default:
    brokers:
    - localhost:9093
    sasl:
      enabled: true
      mechanism: scram-sha512
      username: ${CLIENT_USERNAME}
      password: ${CLIENT_PASSWORD}
EOL
  enrich_kafkactl_yml_with_ssl_configs
}

function prepare_unsecured_config_files() {
  cat > ${KAFKA_HOME}/bin/adminclient.properties << EOL
security.protocol=${SECURITY_PROTOCOL}
sasl.mechanism=GSSAPI
EOL
  enrich_adminclient_properties_with_ssl_configs
  cat > ${KAFKA_HOME}/bin/kcat.properties << EOL
request.timeout.ms=$((${READINESS_PERIOD}*1000))
security.protocol=${SECURITY_PROTOCOL}
EOL
  enrich_kcat_properties_with_ssl_configs
  cat > ${KAFKA_HOME}/bin/kafkactl.yml << EOL
current-context: default
contexts:
  default:
    brokers:
    - localhost:9093
    sasl:
      enabled: false
EOL
  enrich_kafkactl_yml_with_ssl_configs
}

# Prepare Cruise Control Metric Reporter configuration

if [[ "$METRIC_COLLECTOR_ENABLED" == "true" ]]; then
  export CONF_KAFKA_CRUISE_CONTROL_METRICS_REPORTER_SECURITY_PROTOCOL=${SECURITY_PROTOCOL}
  export CONF_KAFKA_METRIC_REPORTERS="com.linkedin.kafka.cruisecontrol.metricsreporter.CruiseControlMetricsReporter"
  if [[ "$DISABLE_SECURITY" == false ]]; then
    export CONF_KAFKA_CRUISE_CONTROL_METRICS_REPORTER_SASL_JAAS_CONFIG="org.apache.kafka.common.security.scram.ScramLoginModule required username=\"${CLIENT_USERNAME}\" password=\"${CLIENT_PASSWORD}\";"
    export CONF_KAFKA_CRUISE_CONTROL_METRICS_REPORTER_SASL_MECHANISM=SCRAM-SHA-512
  fi
  if [[ "${ENABLE_SSL}" == "true" ]]; then
      export CONF_KAFKA_CRUISE_CONTROL_METRICS_REPORTER_BOOTSTRAP_SERVERS=localhost:9092
      export CONF_KAFKA_CRUISE_CONTROL_METRICS_REPORTER_SSL_TRUSTSTORE_LOCATION=${SSL_TRUSTSTORE_LOCATION}
      export CONF_KAFKA_CRUISE_CONTROL_METRICS_REPORTER_SSL_TRUSTSTORE_PASSWORD=changeit
      if [[ "${ENABLE_2WAY_SSL}" == "true" ]]; then
        export CONF_KAFKA_CRUISE_CONTROL_METRICS_REPORTER_SSL_KEYSTORE_LOCATION=${SSL_KEYSTORE_LOCATION}
        export CONF_KAFKA_CRUISE_CONTROL_METRICS_REPORTER_SSL_KEYSTORE_PASSWORD=changeit
        export CONF_KAFKA_CRUISE_CONTROL_METRICS_REPORTER_SSL_KEY_PASSWORD=changeit
      fi
    fi

fi

# Prepare config files to use them in scripts
rm -rf \
  ${KAFKA_HOME}/bin/adminclient.properties \
  ${KAFKA_HOME}/bin/kcat.properties \
  ${KAFKA_HOME}/bin/kafkactl.yml
if [[ "$DISABLE_SECURITY" == false ]]; then
  prepare_secured_config_files
else
  prepare_unsecured_config_files
fi
rm -rf "${KAFKA_HOME}/bin/kafkacat.properties"
cp "${KAFKA_HOME}/bin/kcat.properties" "${KAFKA_HOME}/bin/kafkacat.properties"

if [[ -f ${CONF_KAFKA_LOG_DIRS}/.lock ]]; then
    echo "WARNING: There is FS lock file from previous Kafka pod. Removing it."
    rm "${CONF_KAFKA_LOG_DIRS}/.lock"
fi

# WA for https://issues.apache.org/jira/browse/KAFKA-9444
if [[ -f ${CONF_KAFKA_LOG_DIRS}/meta.properties ]]; then
    echo "WARNING: There is meta.properties file. Removing it."
    rm "${CONF_KAFKA_LOG_DIRS}/meta.properties"
fi

if [[ ${SCAN_FILE_SYSTEM} == "true" ]]; then
    echo "Reading file system before starting."
    ls -Ra "${KAFKA_DATA_DIRS}" > /dev/null 2>&1
fi

# It is necessary to log failed connection events from Kafka in CEF format because Kafka Security does not have access to these events
if [[ ${ENABLE_AUDIT_LOGS} == "true" ]]; then
    cat >> "${KAFKA_HOME}/config/log4j.properties" << EOL
log4j.appender.AUDIT=org.apache.log4j.ConsoleAppender
log4j.appender.AUDIT.layout=org.apache.log4j.PatternLayout
log4j.appender.AUDIT.layout.ConversionPattern = [%d{ISO8601}][%p] [category=kafka.audit] CEF:1|Qubership|Kafka|3.9.1|AUTHENTICATION|%m|1|result=failed type=audit_log_type%n

log4j.logger.org.apache.kafka.common.network.Selector=INFO, AUDIT
log4j.additivity.org.apache.kafka.common.network.Selector=true
EOL
fi

: ${ENABLE_GC_LOGS:="false"}
if [[ "${ENABLE_GC_LOGS}" == "true" ]]; then
  export KAFKA_GC_LOG_OPTS="-Xlog:gc*:file=$KAFKA_HOME/logs/kafkaServer-gc.log:time,tags:filecount=2,filesize=50M"
else
  export KAFKA_GC_LOG_OPTS="-Dnogclog"
fi

# Process the argument to this container ...
case $1 in
  start)
    #
    # Configure the log files ...
    #
    if [[ -z "$LOG_LEVEL" ]]; then
        LOG_LEVEL="INFO"
    fi
    sed -i -r -e "s|=INFO, stdout|=$LOG_LEVEL, stdout|g" ${KAFKA_HOME}/config/log4j.properties
    sed -i -r -e "s|^(log4j.appender.stdout.threshold)=.*|\1=${LOG_LEVEL}|g" ${KAFKA_HOME}/config/log4j.properties
    export KAFKA_LOG4J_OPTS="-Dlog4j.configuration=file:$KAFKA_HOME/config/log4j.properties"
    unset LOG_LEVEL
    # Remove .kafka_started file before Kafka start
    if [[ -f /opt/kafka/.kafka_started ]]; then
      rm -f /opt/kafka/.kafka_started
    fi
    #
    # Process all environment variables that start with 'CONF_KAFKA_':
    #
    for VAR in `env`; do
      env_var=`echo "$VAR" | sed -r "s/(.*)=.*/\1/g"`
      if [[ ${env_var} =~ ^CONF_KAFKA_ ]]; then
        prop_name=`echo "$VAR" | sed -r "s/^CONF_KAFKA_(.*)=.*/\1/g" | tr '[:upper:]' '[:lower:]' | tr _ .`
        if egrep -q "(^|^#)$prop_name=" ${KAFKA_HOME}/config/server.properties; then
          # Note that no config names or values may contain an '@' char
          sed -r -i "s@(^|^#)($prop_name)=(.*)@\2=${!env_var}@g" ${KAFKA_HOME}/config/server.properties
        else
          #echo "Adding property $prop_name=${!env_var}"
          echo "$prop_name=${!env_var}" >> ${KAFKA_HOME}/config/server.properties
        fi
      fi
    done
    if [[ "$KRAFT_ENABLED" == "true" ]]; then
      ${KAFKA_HOME}/bin/kafka-storage.sh format -t "${KRAFT_CLUSTER_ID}" -c "${KAFKA_HOME}/config/server.properties" ${KAFKA_CREDENTIALS}
    fi
    exec ${KAFKA_HOME}/bin/kafka-server-start.sh ${KAFKA_HOME}/config/server.properties
    ;;
esac

# Otherwise just run the specified command
exec "$@"
