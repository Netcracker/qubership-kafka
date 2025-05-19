#!/usr/bin/env bash

/opt/kafka/bin/zookeeper-shell.sh -zk-tls-config-file /opt/kafka/bin/zk-tls-config.properties ${ZOOKEEPER_CONNECT} get /cluster/id | grep id | jq -r '.id'