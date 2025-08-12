#!/bin/bash

# Exit immediately if a *pipeline* returns a non-zero status. (Add -x for command tracing)
set -e
if [[ "$DEBUG" == true ]]; then
  set -x
  printenv
fi

for plugin in "$KAFKA_PLUGINS_DIR"/*; do
  if [[ -d "$plugin" ]]; then
    CLASSPATH="$plugin/*":"$CLASSPATH"
  else
    CLASSPATH="$plugin":"$CLASSPATH"
  fi
done
export CLASSPATH=${CLASSPATH%:}

if [[ -n "$HEAP_OPTS" ]]; then
  export KAFKA_HEAP_OPTS=${HEAP_OPTS}
  unset HEAP_OPTS
fi



cp -f ${KAFKA_HOME}/config/jmx-exporter-config.yml ${KAFKA_HOME}/config/jmx-exporter.yml
if [[ -n "$PROMETHEUS_PORT" ]]; then
  export KAFKA_OPTS="${KAFKA_OPTS} -javaagent:/opt/kafka/libs/prometheus-jmx-exporter-1.1.0.jar=$PROMETHEUS_PORT:${KAFKA_HOME}/config/jmx-exporter.yml"
fi

# Configure internal config providers
export CONF_CONFIG_PROVIDERS="secret"
export ${CLUSTER^^}_CONF_CONFIG_PROVIDERS="secret"
export CONF_CONFIG_PROVIDERS_SECRET_CLASS="io.strimzi.kafka.KubernetesSecretConfigProvider"

OLD_IFS=$IFS # save internal field separator
IFS=", "
read -a clusters <<< "$CLUSTERS"
IFS=${OLD_IFS}

secret_name="${HOSTNAME#"$CLUSTER-"}-secret"
for cluster_name in ${clusters[@]}; do
  cluster_name="${cluster_name//-/_}"
  upper_cluster_name=${cluster_name^^}
  env_kafka_username=${upper_cluster_name}_KAFKA_USERNAME
  env_kafka_password=${upper_cluster_name}_KAFKA_PASSWORD
  env_sasl_mechanism=${upper_cluster_name}_SASL_MECHANISM
  sasl_mechanism=${!env_sasl_mechanism}
  enable_ssl=${upper_cluster_name}_ENABLE_SSL
  security_protocol="PLAINTEXT"
  if [[ "${sasl_mechanism}" == "" ]]; then
    sasl_mechanism="SCRAM-SHA-512"
  fi
  if [[ -n "${!env_kafka_username}" && -n "${!env_kafka_password}" ]]; then
    if [[ "${!enable_ssl}" == "true" ]]; then
      security_protocol="SASL_SSL"
    else
      security_protocol="SASL_PLAINTEXT"
    fi

    key_kafka_username=${cluster_name,,}-kafka-username
    key_kafka_password=${cluster_name,,}-kafka-password
    if [[ "${sasl_mechanism}" == "SCRAM-SHA-512" ]]; then
      export ${upper_cluster_name}_CONF_SASL_JAAS_CONFIG="org.apache.kafka.common.security.scram.ScramLoginModule required \
        username=\"\${secret:${secret_name}:${key_kafka_username}}\" \
        password=\"\${secret:${secret_name}:${key_kafka_password}}\";"
    elif [[ "${sasl_mechanism}" == "PLAIN" ]]; then
      export ${upper_cluster_name}_CONF_SASL_JAAS_CONFIG="org.apache.kafka.common.security.plain.PlainLoginModule required \
        username=\"\${secret:${secret_name}:${key_kafka_username}}\" \
        password=\"\${secret:${secret_name}:${key_kafka_password}}\";"
    fi

    export ${upper_cluster_name}_CONF_SASL_MECHANISM="${sasl_mechanism}"

    unset ${env_kafka_username}
    unset ${env_kafka_password}
  elif [[ ${!enable_ssl} == "true" ]]; then
    security_protocol="SSL"
  fi

  export ${upper_cluster_name}_CONF_SECURITY_PROTOCOL=${security_protocol}

  if [[ "${!enable_ssl}" == "true" ]]; then
    echo "Configuring TLS on ${cluster_name} cluster..."
    kafka_tls_dir=${KAFKA_HOME}/tls/${cluster_name}
    if [[ -f "${kafka_tls_dir}/ca.crt" ]]; then
      kafka_tls_ks_dir=${KAFKA_HOME}/tls-ks/${cluster_name}
      mkdir -p ${kafka_tls_ks_dir}

      if [[ -f "${kafka_tls_dir}/tls.key" && -f "${kafka_tls_dir}/tls.crt" ]]; then
        echo "Preparing keystore"
        # preparing keystore
        openssl pkcs12 -export -out ${kafka_tls_ks_dir}/kafka.keystore.p12 -inkey ${kafka_tls_dir}/tls.key -in ${kafka_tls_dir}/tls.crt -passout pass:changeit
        keytool -importkeystore -destkeystore ${kafka_tls_ks_dir}/kafka.keystore.jks -deststorepass changeit -srcstoretype PKCS12 -srckeystore ${kafka_tls_ks_dir}/kafka.keystore.p12 -srcstorepass changeit
        keytool -import -trustcacerts -keystore ${kafka_tls_ks_dir}/kafka.keystore.jks -storepass changeit -noprompt -alias ca-cert -file ${kafka_tls_dir}/ca.crt

        export ${upper_cluster_name}_CONF_SSL_KEYSTORE_LOCATION=${kafka_tls_ks_dir}/kafka.keystore.jks
        export ${upper_cluster_name}_CONF_SSL_KEYSTORE_PASSWORD=changeit
        export ${upper_cluster_name}_CONF_SSL_KEY_PASSWORD=changeit
        export ${upper_cluster_name}_CONF_CONSUMER_SSL_KEYSTORE_LOCATION=${kafka_tls_ks_dir}/kafka.keystore.jks
        export ${upper_cluster_name}_CONF_CONSUMER_SSL_KEYSTORE_PASSWORD=changeit
        export ${upper_cluster_name}_CONF_CONSUMER_SSL_KEY_PASSWORD=changeit
        export ${upper_cluster_name}_CONF_PRODUCER_SSL_KEYSTORE_LOCATION=${kafka_tls_ks_dir}/kafka.keystore.jks
        export ${upper_cluster_name}_CONF_PRODUCER_SSL_KEYSTORE_PASSWORD=changeit
        export ${upper_cluster_name}_CONF_PRODUCER_SSL_KEY_PASSWORD=changeit

      fi

      echo "Preparing truststore"
      # preparing truststore
      keytool -keystore ${kafka_tls_ks_dir}/kafka.truststore.jks -alias ca -import -trustcacerts -file ${kafka_tls_dir}/ca.crt -storepass changeit -noprompt

      export ${upper_cluster_name}_CONF_SSL_TRUSTSTORE_LOCATION=${kafka_tls_ks_dir}/kafka.truststore.jks
      export ${upper_cluster_name}_CONF_SSL_TRUSTSTORE_PASSWORD=changeit
      export ${upper_cluster_name}_CONF_CONSUMER_SSL_TRUSTSTORE_LOCATION=${kafka_tls_ks_dir}/kafka.truststore.jks
      export ${upper_cluster_name}_CONF_CONSUMER_SSL_TRUSTSTORE_PASSWORD=changeit
      export ${upper_cluster_name}_CONF_PRODUCER_SSL_TRUSTSTORE_LOCATION=${kafka_tls_ks_dir}/kafka.truststore.jks
      export ${upper_cluster_name}_CONF_PRODUCER_SSL_TRUSTSTORE_PASSWORD=changeit
      export ${upper_cluster_name}_CONF_SSL_ENDPOINT_IDENTIFICATION_ALGORITHM=""
    fi

    echo "TLS configuration is applied"
  fi
done

mkdir -p "${KAFKA_HOME}/dumps"
export KAFKA_OPTS="-XX:+HeapDumpOnOutOfMemoryError -XX:HeapDumpPath=${KAFKA_HOME}/dumps/ -XX:+ExitOnOutOfMemoryError ${KAFKA_OPTS}"

# Process the argument to this container ...
case $1 in
  start)
    #
    # Configure the log files ...
    #
    if [[ -z "$LOG_LEVEL" ]]; then
        LOG_LEVEL="INFO"
    fi
    sed -i -r -e "s|=INFO, stdout|=$LOG_LEVEL, stdout|g" ${KAFKA_HOME}/config/connect-log4j.properties
    sed -i -r -e "s|^(log4j.appender.stdout.threshold)=.*|\1=${LOG_LEVEL}|g" ${KAFKA_HOME}/config/connect-log4j.properties
    export KAFKA_LOG4J_OPTS="-Dlog4j.configuration=file:$KAFKA_HOME/config/connect-log4j.properties"
    unset LOG_LEVEL

    #
    # Generate mm2 configuration only if it doesn't exist
    #
    additional_log=""
    if [[ -f "${KAFKA_HOME}/config/kmm/kmm.conf" ]]; then
      cat ${KAFKA_HOME}/config/kmm/kmm.conf > ${KAFKA_HOME}/config/mm2.properties
      echo "" >> ${KAFKA_HOME}/config/mm2.properties
      additional_log="additional "
    else
      echo "# mm2.properties" > ${KAFKA_HOME}/config/mm2.properties
      echo "clusters = $CLUSTERS" >> ${KAFKA_HOME}/config/mm2.properties
      echo "" >> ${KAFKA_HOME}/config/mm2.properties

      echo "# configure a specific source->target replication flow" >> ${KAFKA_HOME}/config/mm2.properties
      for source_cluster_name in ${clusters[@]}; do
        for target_cluster_name in ${clusters[@]}; do
          if [[ ${source_cluster_name} != ${target_cluster_name} ]]; then
            echo "${source_cluster_name}->${target_cluster_name}.enabled = true" >> ${KAFKA_HOME}/config/mm2.properties
          fi
        done
      done
    fi

    echo "" >> ${KAFKA_HOME}/config/mm2.properties
    echo "# configure ${additional_log}common properties" >> ${KAFKA_HOME}/config/mm2.properties
    for VAR in `env`; do
      env_var=`echo "$VAR" | sed -r "s/(.*)=.*/\1/g"`
      if [[ ${env_var} =~ ^CONF_ ]]; then
        prop_name=`echo "$VAR" | sed -r "s/^CONF_(.*)=.*/\1/g" | tr '[:upper:]' '[:lower:]' | tr _ .`
        if egrep -q "(^|^#)$prop_name = " ${KAFKA_HOME}/config/mm2.properties; then
          # Note that no config names or values may contain an '@' char
          sed -r -i "s@(^|^#)($prop_name) = (.*)@\2 = ${!env_var}@g" ${KAFKA_HOME}/config/mm2.properties
        else
          echo "$prop_name = ${!env_var}" >> ${KAFKA_HOME}/config/mm2.properties
        fi
      fi
    done

    for cluster_name in ${clusters[@]}; do
      echo "" >> ${KAFKA_HOME}/config/mm2.properties
      echo "# configure ${additional_log}properties for [$cluster_name] cluster" >> ${KAFKA_HOME}/config/mm2.properties
      for VAR in `env`; do
        env_var=`echo "$VAR" | sed -r "s/(.*)=.*/\1/g"`
        if [[ ${env_var} =~ ^${cluster_name^^}_CONF_ ]]; then
          prop_name=`echo "$VAR" | sed -r "s/^${cluster_name^^}_CONF_(.*)=.*/${cluster_name^^}_\1/g" | tr '[:upper:]' '[:lower:]' | tr _ .`
          if egrep -q "(^|^#)$prop_name = " ${KAFKA_HOME}/config/mm2.properties; then
            # Note that no config names or values may contain an '@' char
            sed -r -i "s@(^|^#)($prop_name) = (.*)@\2 = ${!env_var}@g" ${KAFKA_HOME}/config/mm2.properties
          else
            echo "$prop_name = ${!env_var}" >> ${KAFKA_HOME}/config/mm2.properties
          fi
        fi
      done
    done

    echo "" >> ${KAFKA_HOME}/config/mm2.properties
    echo "# configure connectors properties" >> ${KAFKA_HOME}/config/mm2.properties
    for source_cluster_name in ${clusters[@]}; do
      for target_cluster_name in ${clusters[@]}; do
        if [[ ${source_cluster_name} != ${target_cluster_name} ]]; then
          for VAR in `env`; do
            env_var=`echo "$VAR" | sed -r "s/(.*)=.*/\1/g"`
            if [[ ${env_var} =~ ^${source_cluster_name^^}_${target_cluster_name^^}_CONF_ ]]; then
              prop_name=`echo "$VAR" | sed -r "s/^${source_cluster_name^^}_${target_cluster_name^^}_CONF_(.*)=.*/${source_cluster_name^^}->${target_cluster_name^^}_\1/g" | tr '[:upper:]' '[:lower:]' | tr _ .`
              if egrep -q "(^|^#)$prop_name = " ${KAFKA_HOME}/config/mm2.properties; then
                # Note that no config names or values may contain an '@' char
                sed -r -i "s@(^|^#)($prop_name) = (.*)@\2 = ${!env_var}@g" ${KAFKA_HOME}/config/mm2.properties
              else
                echo "$prop_name = ${!env_var}" >> ${KAFKA_HOME}/config/mm2.properties
              fi
            fi
          done
        fi
      done
    done

    exec ${KAFKA_HOME}/bin/connect-mirror-maker.sh ${KAFKA_HOME}/config/mm2.properties --clusters ${CLUSTER}
    ;;
esac

# Otherwise just run the specified command
exec "$@"