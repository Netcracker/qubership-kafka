# Declarative Users Management

## Introduction

This section describes the `Kafka Users Configurator`. The `Kafka Users Configurator` is part of the
Kafka Service Operator that is responsible for applying and deleting `KafkaUser` configs. The `KafkaUser`
specifies the Kafka user to be created according to provided `Secret` with user credentials to apply,
role that describes access provided to user and Kafka cluster, for which such user need to be applied.

## KafkaUser custom resource overview

To create Kafka user declaratively using `Kafka Users Configurator` client service should apply
`KafkaUser` Kubernetes Custom Resource. This is a common example:

```yaml
apiVersion: qubership.org/v1
kind: KafkaUser
metadata:
  name: kafka-user
  namespace: kafka-service
  annotations:
    kafka.qubership.org/bootstrap.servers: kafka.kafka-service:9092
spec:
  authentication:
    type: scram-sha-512
    secret:
      name: kafka-user-secret
      format: connection-properties
      generate: false
  authorization:
    role: namespace-admin
```

Where:

* `authentication.type` describes the type of authentication for created user. It can be `scram-sha-512` only.
* `authentication.secret.generate` describes whether to create Kubernetes secret by operator or it 
should be pre-created on the application side.
* `authentication.secret.name` is the name of Kubernetes secret where Kafka credentials are stored. 
If `authentication.secret.generate` is disabled this secret should be created before CR.
* `authentication.secret.format` describes the format of Kafka connection config. It can be 
`connection-properties` (the set of `username`, `password` parameters specified) or `connection-uri` 
where all parameters specified in one-line format (`kafka://username:password@address:port`).
* `authentication.watchSecret` describes whether to use `authentication.secret` to apply `KafkaUser`
credentials or keep previously defined. If set to `false`, secret can be detached from `KafkaUser`
custom resource and will not be updated during reconciliation. By default, it is set to `true`.
* `authorization.role` describes the set of `ACL` resources applied for created Kafka user. It can be
`admin` (access to all resources in cluster), `namespace-admin` (access to resources with namespace prefix).

If `authentication.secret.generate` is disabled the secret need to be pre-created to apply `KafkaUser`.

Example of secret in `connection-properties` format:

```yaml
kind: Secret
apiVersion: v1
metadata:
  name: kafka-user-secret
  namespace: kafka-service
data:
  bootstrap.servers: "kafka.kafka-service:9092"
  security.protocol: "sasl_plaintext"
  sasl.mechanism: "scram-sha-512"
  username: kafka-service_kafka-user
  password: admin
```

Example of secret in `connection-uri` format:

```yaml
kind: Secret
apiVersion: v1
metadata:
  name: kafka-user-secret
  namespace: kafka-service
data:
  connection-uri: kafka://kafka-service_kafka-user:password@kafka.kafka-service:9092?security.protocol=sasl_plaintext&sasl.mechanism=scram-sha-512
```

**Note**: The Secret for `KafkaUser` must be created in the same namespace as `KafkaUser` Custom Resource.

Kafka Service Operator processes all caught `KafkaUser` Custom Resources that are placed in namespaces
specified in `operator.kafkaUserConfigurator.watchNamespace` parameter and have the same Kafka address 
specified in `kafka.qubership.org/bootstrap.servers` annotation as that Kafka Service Operator uses.

## KafkaUser custom resource validation

`KafkaUser` is invalid if the `username` specified in `authentication.secret.name` secret is not 
in format `<cr.namespace>_<cr.name>` where `cr.namespace` and `cr.name` are namespace and name of 
`KafkaUser` custom resource.
