#!/usr/bin/env bash

source_config="/opt/docker/src/application.conf"
target_config="/opt/docker/conf/application.conf"
cp "${source_config}" "${target_config}"

: ${KAFKA_SASL_MECHANISM:="SCRAM-SHA-512"}

SECURITY_PROTOCOL="PLAINTEXT"
if [[ "${KAFKA_ENABLE_SSL}" == "true" ]]; then
  SECURITY_PROTOCOL="SSL"
  kafka_ca_cert_path="/tls/ca.crt"
  kafka_tls_key_path="/tls/tls.key"
  kafka_tls_cert_path="/tls/tls.crt"
  kafka_tls_ks_dir="${LAG_EXPORTER_HOME}/tls-ks"
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
    fi

    SSL_TRUSTSTORE_LOCATION="${kafka_tls_ks_dir}/kafka.truststore.jks"
    keytool -import -trustcacerts -keystore ${SSL_TRUSTSTORE_LOCATION} -storepass ${SSL_PASSWORD} -noprompt -alias ca -file $kafka_ca_cert_path

    SSL_CONFIGURATION="ssl.truststore.location = \"${SSL_TRUSTSTORE_LOCATION}\"\n        ssl.truststore.password = \"${SSL_PASSWORD}\"\n        ${SSL_CONFIGURATION}"
  fi
fi

if [[ -n ${KAFKA_USER} && -n ${KAFKA_PASSWORD} ]]; then
  SASL_LOGIN_MODULE="org.apache.kafka.common.security.scram.ScramLoginModule"
  if [[ "${KAFKA_SASL_MECHANISM}" == "PLAIN" ]]; then
      SASL_LOGIN_MODULE="org.apache.kafka.common.security.plain.PlainLoginModule"
  fi
  SASL_CONFIGURATION="security.protocol = \"SASL_${SECURITY_PROTOCOL}\"\n        sasl.mechanism = \"${KAFKA_SASL_MECHANISM}\"\n        sasl.jaas.config = \"${SASL_LOGIN_MODULE} required username=\\\\\"${KAFKA_USER}\\\\\" password=\\\\\"${KAFKA_PASSWORD}\\\\\";\""
else
  SASL_CONFIGURATION="security.protocol = \"${SECURITY_PROTOCOL}\""
fi

CONFIGURATION="${SASL_CONFIGURATION}\n        ${SSL_CONFIGURATION}"
sed -i "s%\${CONFIGURATION}%${CONFIGURATION}%g" ${target_config}

/opt/docker/bin/kafka-lag-exporter "-Dconfig.file=${target_config}" "-Dlogback.configurationFile=/opt/docker/conf/logback.xml"