#!/bin/bash

# Initialise writable runtime directories under /tmp so the container can run
# with readOnlyRootFilesystem: true.  Static CC config is in /cruise-control/config;
# we copy it plus the UI bundle to /tmp/cc at each start.
CC_WORK=/tmp/cc
export KAFKA_CTL_CONFIG="${CC_WORK}/kafkactl.yml"
mkdir -p "${CC_WORK}/config" "${CC_WORK}/tls-ks"
cp -r /cruise-control/config/. "${CC_WORK}/config/"
if [[ "${UI_ENABLED}" != "false" ]]; then
  cp -r /cruise-control/cruise-control-ui "${CC_WORK}/cruise-control-ui"
fi

# Resolve security protocol.
#
#$1 - security protocol without SSL
#$2 - security protocol with SSL
prepare_secured_client() {
echo "sasl.jaas.config=org.apache.kafka.common.security.scram.ScramLoginModule required username=\"${KAFKA_AUTH_USERNAME}\" password=\"${KAFKA_AUTH_PASSWORD}\";" >> ${CC_WORK}/config/cruisecontrol.properties
cat > ${CC_WORK}/config/cruise_control_jaas.conf <<EOL
KafkaClient {
  org.apache.kafka.common.security.scram.ScramLoginModule required
  username="${KAFKA_AUTH_USERNAME}"
  password="${KAFKA_AUTH_PASSWORD}"
};
EOL
}

function enrich_kafkactl_yml_with_ssl_configs() {
  if [[ "${KAFKA_ENABLE_SSL}" == "true" && -f "/tls/ca.crt" ]]; then
    cat >> "${KAFKA_CTL_CONFIG}" << EOL
    tls:
      enabled: true
      ca: /tls/ca.crt
EOL
    if [[ -f "/tls/tls.crt" && -f "/tls/tls.key" ]]; then
      cat >> ${KAFKA_CTL_CONFIG} << EOL
      cert: /tls/tls.crt
      certKey: /tls/tls.key
EOL
    fi
  fi
}

function prepare_secured_config_files() {
  cat > "${KAFKA_CTL_CONFIG}" << EOL
current-context: default
contexts:
  default:
    brokers: [${BOOTSTRAP_SERVERS}]
    sasl:
      enabled: true
      mechanism: scram-sha512
      username: ${KAFKA_AUTH_USERNAME}
      password: ${KAFKA_AUTH_PASSWORD}
EOL
}

function prepare_unsecured_config_files() {
  cat > "${KAFKA_CTL_CONFIG}" << EOL
current-context: default
contexts:
  default:
    brokers: [${BOOTSTRAP_SERVERS}]
    sasl:
      enabled: false
EOL
}

# Prepare main config file
cat >> ${CC_WORK}/config/cruisecontrol.properties <<EOF
bootstrap.servers=$BOOTSTRAP_SERVERS
self.healing.enabled=$SELF_HEALING_ENABLED
capacity.config.file=config/capacityConfigFile.json
kafka.broker.failure.detection.enable=true
webserver.auth.credentials.file=config/ui-credentianals
webserver.security.enable=true
EOF

if [[ "${KAFKA_ENABLE_SSL}" == "true" && -n "${KAFKA_AUTH_USERNAME}" && -n "${KAFKA_AUTH_PASSWORD}" ]]; then
  echo 'security.protocol=SASL_SSL' >> ${CC_WORK}/config/cruisecontrol.properties
  echo 'sasl.mechanism=SCRAM-SHA-512' >> ${CC_WORK}/config/cruisecontrol.properties
  prepare_secured_client
elif [[ -n "${KAFKA_AUTH_USERNAME}" && -n "${KAFKA_AUTH_PASSWORD}" ]]; then
  echo 'security.protocol=SASL_PLAINTEXT' >> ${CC_WORK}/config/cruisecontrol.properties
  echo 'sasl.mechanism=SCRAM-SHA-512' >> ${CC_WORK}/config/cruisecontrol.properties
  prepare_secured_client
elif [[ "${KAFKA_ENABLE_SSL}" == "true" ]]; then
  echo 'security.protocol=SSL' >> ${CC_WORK}/config/cruisecontrol.properties
else
  echo 'security.protocol=PLAINTEXT' >> ${CC_WORK}/config/cruisecontrol.properties
fi

#Prepare kafkactl config

if [[ -n ${KAFKA_AUTH_USERNAME} && -n ${KAFKA_AUTH_PASSWORD} ]]; then
  prepare_secured_config_files
else
  prepare_unsecured_config_files
fi
enrich_kafkactl_yml_with_ssl_configs


#Prepare UI credentianals file
cat > ${CC_WORK}/config/ui-credentianals << EOF
$ADMIN_USERNAME: $ADMIN_PASSWORD,ADMIN
EOF
if [[ -n "${VIEWER_USERNAME}" && -n "${VIEWER_PASSWORD}" ]]; then
cat >> ${CC_WORK}/config/ui-credentianals << EOF
$VIEWER_USERNAME: $VIEWER_PASSWORD,VIEWER
EOF
fi

#Prepare UI endpoint file (only if UI is enabled; the dir was copied in the init block above)
if [[ "${UI_ENABLED}" != "false" ]]; then
  cat > ${CC_WORK}/cruise-control-ui/dist/static/config.csv <<EOL
$CLUSTER_NAME,$CLUSTER_NAME,/kafkacruisecontrol/
EOL
fi

#Prepare broker's capacity config file
: ${BROKER_DISK_SPACE:=10000} # In MB
: ${BROKER_NW_IN:=10000} # in KB
: ${BROKER_NW_OUT:=10000} # in KB
: ${BROKER_CPU:=100} # In Percentage
cat >> ${CC_WORK}/config/capacityConfigFile.json << EOL
{"brokerCapacities":[
{"brokerId": "-1","capacity": {"DISK": "$BROKER_DISK_SPACE","CPU": "$BROKER_CPU","NW_IN": "$BROKER_NW_IN","NW_OUT": "$BROKER_NW_OUT"}}
]}
EOL

if [[ "${KAFKA_ENABLE_SSL}" == "true" ]]; then
    kafka_ca_cert_path="tls/ca.crt"
    kafka_tls_key_path="tls/tls.key"
    kafka_tls_cert_path="tls/tls.crt"
    kafka_tls_ks_dir="${CC_WORK}/tls-ks"
    mkdir -p ${kafka_tls_ks_dir}
    if [[ -f $kafka_ca_cert_path ]]; then
    SSL_PASSWORD="changeit"
    if [[ -f $kafka_tls_key_path && -f $kafka_tls_cert_path ]]; then
      SSL_KEYSTORE_LOCATION="${kafka_tls_ks_dir}/kafka.keystore.jks"
      openssl pkcs12 -export -in $kafka_tls_cert_path -inkey $kafka_tls_key_path -out ${kafka_tls_ks_dir}/kafka.keystore.p12 -passout pass:${SSL_PASSWORD}
      keytool -importkeystore -destkeystore ${SSL_KEYSTORE_LOCATION} -deststorepass ${SSL_PASSWORD} \
        -srckeystore ${kafka_tls_ks_dir}/kafka.keystore.p12 -srcstoretype PKCS12 -srcstorepass ${SSL_PASSWORD}
      keytool -import -trustcacerts -keystore ${SSL_KEYSTORE_LOCATION} -storepass ${SSL_PASSWORD} -noprompt -alias ca-cert -file $kafka_ca_cert_path
      SSL_CONFIGURATION="ssl.keystore.location = \"${SSL_KEYSTORE_LOCATION}\"\n        ssl.keystore.password = \"${SSL_PASSWORD}\"\n        ssl.key.password = \"${SSL_PASSWORD}\""
    cat >> ${CC_WORK}/config/cruisecontrol.properties <<EOF
ssl.keystore.type=JKS
ssl.keystore.location=$SSL_KEYSTORE_LOCATION
ssl.keystore.password=${SSL_PASSWORD}
ssl.key.password=${SSL_PASSWORD}
EOF
    fi

    SSL_TRUSTSTORE_LOCATION="${kafka_tls_ks_dir}/kafka.truststore.jks"
    keytool -import -trustcacerts -keystore ${SSL_TRUSTSTORE_LOCATION} -storepass ${SSL_PASSWORD} -noprompt -alias ca -file $kafka_ca_cert_path
    cat >> ${CC_WORK}/config/cruisecontrol.properties <<EOF
ssl.truststore.type=JKS
ssl.truststore.location=$SSL_TRUSTSTORE_LOCATION
ssl.truststore.password=${SSL_PASSWORD}
EOF
    fi
fi

# Process the argument to this container ...
for VAR in `env`; do
  env_var=`echo "$VAR" | sed -r "s/(.*)=.*/\1/g"`
  if [[ ${env_var} =~ ^CONF_ ]]; then
    prop_name=`echo "$VAR" | sed -r "s/^CONF_(.*)=.*/\1/g" | tr '[:upper:]' '[:lower:]' | tr _ .`
    if egrep -q "(^|^#)$prop_name=" ${CC_WORK}/config/cruisecontrol.properties; then
      # Note that no config names or values may contain an '@' char
      sed -r -i "s@(^|^#)($prop_name)=(.*)@\2=${!env_var}@g" ${CC_WORK}/config/cruisecontrol.properties
    else
      #echo "Adding property $prop_name=${!env_var}"
      echo "$prop_name=${!env_var}" >> ${CC_WORK}/config/cruisecontrol.properties
    fi
  fi
done

if [[ "${EXTERNAL_KAFKA_ENABLED}" == "false" ]]; then
  kafkactl create topic __CruiseControlMetrics --partitions=-1 --replication-factor=-1
fi

# Write config map to main config file
cat /cruise-control/additionalConfig/cruisecontrolAdditionalProperties.conf >> ${CC_WORK}/config/cruisecontrol.properties

# Point webserver to the runtime UI path (or disable it)
if [[ "${UI_ENABLED}" != "false" ]]; then
  echo "webserver.ui.diskpath=${CC_WORK}/cruise-control-ui/dist" >> ${CC_WORK}/config/cruisecontrol.properties
fi

/cruise-control/kafka-cruise-control-start.sh ${CC_WORK}/config/cruisecontrol.properties 9090



