This guide describes how to migrate Kafka service from ZooKeeper to Kraft. 
For more information about migration see [apache guide](https://kafka.apache.org/documentation/#kraft_zk_migration)

[[_TOC_]]

# Automatic migration

To automatically migrate Kafka service from ZooKeeper to Kraft you need to set following deploy parameters and run upgrade job:

* Keep ZooKeeper parameters
* Set `kafka.kraft.enabled: true`
* Set `kafka.kraft.migration: true`
* Set `kafka.migrationController` parameters section

After that you can check migration process in Kafka operator pod logs.

# Manual migration

This section describes how to manually migrate Kafka service from ZooKeeper to Kraft. 

## Initial step

Get `KRAFT_CLUSTER_ID`, to do that from one of Kafka pods use command `${KAFKA_HOME}/bin/get-cluster-id.sh`.

Prepare Kraft controller PVC, deployment and service template, to do that modify following templates:

* Deployment, you need to change ZooKeeper and Kafka parameters according to your Kafka cluster parameters and 
environment parameters (security context, node affinity, etc.):

    <details>
    <summary>Click to expand YAML</summary>
    
    ```yaml
    kind: Deployment
    apiVersion: apps/v1
    metadata:
      name: kafka-kraft-controller
      namespace: ${KAFKA_NAMESPACE}
    spec:
      replicas: 1
      selector:
        matchLabels:
          name: kafka-kraft-controller
      template:
        metadata:
          labels:
            name: kafka-kraft-controller
        spec:
          volumes:
            - name: data
              persistentVolumeClaim:
                claimName: pvc-kafka-kraft-controller
            - name: log
              emptyDir: {}
            - name: trusted-certs
              secret:
                secretName: kafka-trusted-certs
                defaultMode: 420
            - name: public-certs
              secret:
                secretName: kafka-public-certs
                defaultMode: 420
            - name: ssl-certs
              secret:
                secretName: kafka-tls-secret
                defaultMode: 420
          containers:
            - name: kafka
              image: >-
                ${KAFKA_IMAGE}
              ports:
                - containerPort: 9092
                  protocol: TCP
                - containerPort: 9093
                  protocol: TCP
                - containerPort: 9094
                  protocol: TCP
                - containerPort: 9095
                  protocol: TCP
                - containerPort: 9096
                  protocol: TCP
                - containerPort: 9087
                  protocol: TCP
                - containerPort: 8080
                  protocol: TCP
              env:
                - name: BROKER_ID
                  value: '3000'
                - name: VOTERS
                  value: '3000@localhost:9092'
                - name: CONTROLLER_LISTENER_NAMES
                  value: CONTROLLER
                - name: LISTENERS
                  value: 'CONTROLLER://:9092'
                - name: KRAFT_CLUSTER_ID
                  value: '${KRAFT_CLUSTER_ID}'
                - name: INTER_BROKER_LISTENER_NAME
                  value: INTERNAL
                - name: PROCESS_ROLES
                  value: controller
                - name: KRAFT_ENABLED
                  value: 'true'
                - name: MIGRATION_CONTROLLER
                  value: 'true'
                - name: READINESS_PERIOD
                  value: '30'
                - name: REPLICATION_FACTOR
                  value: '3'
                - name: EXTERNAL_HOST_NAME
                - name: EXTERNAL_PORT
                - name: CONF_KAFKA_BROKER_RACK
                - name: INTERNAL_HOST_NAME
                  value: kafka-kraft-controller.kafka-service
                - name: INTER_BROKER_HOST_NAME
                  value: kafka-kraft-controller.kafka-broker.kafka-service
                - name: HEAP_OPTS
                  value: '-Xms256m -Xmx256m'
                - name: DISABLE_SECURITY
                  value: 'false'
                - name: CLOCK_SKEW
                  value: '10'
                - name: JWK_SOURCE_TYPE
                  value: jwks
                - name: JWKS_CONNECTION_TIMEOUT
                  value: '1000'
                - name: TOKEN_ROLES_PATH
                  value: resource_access.account.roles
                - name: JWKS_READ_TIMEOUT
                  value: '1000'
                - name: JWKS_SIZE_LIMIT
                  value: '51200'
                - name: ENABLE_AUDIT_LOGS
                  value: 'false'
                - name: ENABLE_AUTHORIZATION
                  value: 'false'
                - name: HEALTH_CHECK_TIMEOUT
                  value: '30'
                - name: ZOOKEEPER_CONNECT
                  value: zookeeper.zookeeper-service:2181
                - name: ZOOKEEPER_SET_ACL
                  value: 'false'
                - name: ADMIN_USERNAME
                  valueFrom:
                    secretKeyRef:
                      name: kafka-secret
                      key: admin-username
                - name: ADMIN_PASSWORD
                  valueFrom:
                    secretKeyRef:
                      name: kafka-secret
                      key: admin-password
                - name: CLIENT_USERNAME
                  valueFrom:
                    secretKeyRef:
                      name: kafka-secret
                      key: client-username
                - name: CLIENT_PASSWORD
                  valueFrom:
                    secretKeyRef:
                      name: kafka-secret
                      key: client-password
                - name: IDP_WHITELIST
                  valueFrom:
                    secretKeyRef:
                      name: kafka-secret
                      key: idp-whitelist
                - name: ZOOKEEPER_CLIENT_USERNAME
                  valueFrom:
                    secretKeyRef:
                      name: kafka-secret
                      key: zookeeper-client-username
                - name: ZOOKEEPER_CLIENT_PASSWORD
                  valueFrom:
                    secretKeyRef:
                      name: kafka-secret
                      key: zookeeper-client-password
                - name: ENABLE_SSL
                  value: 'true'
                - name: SSL_CIPHER_SUITES
                - name: ENABLE_2WAY_SSL
                  value: 'false'
                - name: ALLOW_NONENCRYPTED_ACCESS
                  value: 'true'
                - name: ENABLE_ZOOKEEPER_SSL
                  value: 'true'
                - name: CONF_KAFKA_AUTO_CREATE_TOPICS_ENABLE
                  value: 'false'
              resources:
                limits:
                  cpu: 400m
                  memory: 800Mi
                requests:
                  cpu: 60m
                  memory: 600Mi
              volumeMounts:
                - name: data
                  mountPath: /var/opt/kafka/data
                - name: log
                  mountPath: /opt/kafka/logs
                - name: trusted-certs
                  mountPath: /opt/kafka/trustcerts
                - name: public-certs
                  mountPath: /opt/kafka/public-certs
                - name: ssl-certs
                  mountPath: /opt/kafka/tls
              livenessProbe:
                exec:
                  command:
                    - ./bin/kafka-health.sh
                initialDelaySeconds: 50
                timeoutSeconds: 5
                periodSeconds: 15
                successThreshold: 1
                failureThreshold: 5
              readinessProbe:
                exec:
                  command:
                    - ./bin/kafka-health.sh
                initialDelaySeconds: 60
                timeoutSeconds: 30
                periodSeconds: 30
                successThreshold: 1
                failureThreshold: 5
              terminationMessagePath: /dev/termination-log
              terminationMessagePolicy: File
              imagePullPolicy: Always
              securityContext:
                capabilities:
                  drop:
                    - ALL
                allowPrivilegeEscalation: false
          restartPolicy: Always
          terminationGracePeriodSeconds: 1800
          dnsPolicy: ClusterFirst
          serviceAccountName: kafka
          serviceAccount: kafka
          securityContext:
            runAsNonRoot: true
            fsGroup: 1000
            seccompProfile:
              type: RuntimeDefault
          hostname: kafka-kraft-controller
          subdomain: kafka-kraft-controller
          affinity: {}
          schedulerName: default-scheduler
      strategy:
        type: Recreate
      revisionHistoryLimit: 10
      progressDeadlineSeconds: 3600
    ```

    </details>
  
* PVC, for storage class or predefined PV:

    <details>
    <summary>Click to expand YAML</summary>
    
    ```yaml
    kind: PersistentVolumeClaim
    apiVersion: v1
    metadata:
      name: pvc-kafka-kraft-controller
      namespace: ${KAFKA_NAMESPACE}
      labels:
        name: kafka-kraft-controller
    spec:
      accessModes:
        - ReadWriteOnce
      resources:
        requests:
          storage: 1Gi
      storageClassName: ${STORAGE_CLASS_NAME}
      volumeMode: Filesystem
    ```

    </details>
  
* Service:

    <details>
    <summary>Click to expand YAML</summary>
    
    ```yaml
    kind: Service
    apiVersion: v1
    metadata:
      name: kafka-kraft-controller
      namespace: ${KAFKA_NAMESPACE}
      labels:
        name: kafka-kraft-controller
    spec:
      ports:
        - name: kafka-kraft-controller
          protocol: TCP
          port: 9092
          targetPort: 9092
      selector:
        name: kafka-kraft-controller
    ```

    </details>
 
If TLS is enabled it is necessary to add `kafka-kraft-controller` to additional DNS names in TLS certificate.
 
## Creating Kraft controller

Apply all templates prepared in previous step and wait until Kraft controller pod starts successfully.

## Updating Kafka brokers to do metadata migration

Add following ENVs to all kafka broker deployments and restart pods:

```yaml
- name: MIGRATION_BROKER
  value: 'true'
- name: VOTERS
  value: '3000@kafka-kraft-controller:9092'
```

Wait until Kafka pods starts and execute in Kraft controller 
pod `${KAFKA_HOME}/bin/get-kraft-migration-status.sh`, result should be `true`.
To check migration logs see file `/opt/kafka/logs/migration.log` inside controller pod.

## Removing ZooKeeper connection from Kafka brokers

Remove ENV `MIGRATION_BROKER` from Kafka broker deployments and add following ENVs:

```yaml
- name: PROCESS_ROLES
  value: broker
- name: KRAFT_ENABLED
  value: 'true'
- name: VOTERS
  value: '3000@kafka-kraft-controller:9092'
- name: MIGRATED_BROKER
  value: 'true'
- name: KRAFT_CLUSTER_ID
  value: '${KRAFT_CLUSTER_ID}'
```

Replace `${KRAFT_CLUSTER_ID}` with correct value and remove all ENVs related to ZooKeeper. 
Wait until all Kafka broker started.

## Removing ZooKeeper connection from Kraft controller

Remove ENV `MIGRATION_CONTROLLER` from Kraft controller deployment and add following ENV:

```yaml
- name: MIGRATED_CONTROLLER
  value: 'true'
```

Remove all ENVs related to ZooKeeper. Wait until Kraft controller started.

## Create Kraft cluster with brokers and controller

To service `kafka-broker` add:

```yaml
- name: kraft-controller
  protocol: TCP
  port: 9096
  targetPort: 9096
```

In all Kafka brokers deployments remove ENV `MIGRATED_BROKER` and replace previous Kraft envs with:

```yaml
- name: PROCESS_ROLES
  value: broker, controller
- name: KRAFT_ENABLED
  value: 'true'
- name: VOTERS
  value: '3000@kafka-kraft-controller:9092,1@kafka-1.kafka-broker.${KAFKA_NAMESPACE}:9096,2@kafka-2.kafka-broker.${KAFKA_NAMESPACE}:9096,3@kafka-3.kafka-broker.${KAFKA_NAMESPACE}:9096'
- name: KRAFT_CLUSTER_ID
  value: '${KRAFT_CLUSTER_ID}'
```

Replace ${KAFKA_NAMESPACE} and ${KRAFT_CLUSTER_ID} with actual values.

To broker ports add:

```yaml
- containerPort: 9096
  protocol: TCP
```

In Kraft controller deployment replace ENV `VOTERS` with:

```yaml
- name: VOTERS
  value: '3000@kafka-kraft-controller:9092,1@kafka-1.kafka-broker.${KAFKA_NAMESPACE}:9096,2@kafka-2.kafka-broker.${KAFKA_NAMESPACE}:9096,3@kafka-3.kafka-broker.${KAFKA_NAMESPACE}:9096'
```

same as in broker deployments. Wait until controller and brokers started.

## Removing Kraft controller

In all Kafka broker deployments replace ENV `VOTERS` with:

```yaml
- name: VOTERS
  value: '1@kafka-1.kafka-broker.${KAFKA_NAMESPACE}:9096,2@kafka-2.kafka-broker.${KAFKA_NAMESPACE}:9096,3@kafka-3.kafka-broker.${KAFKA_NAMESPACE}:9096'
```

Wait until brokers started. Delete Kraft controller deployment, PVC and service. Migration complete.

## Enabling Kraft for future upgrades

Enable Kraft in deploy parameters: set `kafka.kraft.enabled` to `true`

Add label `kraft: enabled` to all Kafka brokers PVCs.
