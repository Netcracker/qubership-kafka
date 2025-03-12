Kafka Service Operator has an ability to manage special kind of Custom Resource (CR), `KmmConfig`.
It allows other applications to define changes for Kafka Mirror Maker config map.
This is helpful to eliminate manual configuration steps after installation.
In `KmmConfig` the Custom Resource application can specify the list of topics or topic's regexp which should be replicated from "source"
Kafka datacenter to "target" Kafka datacenter.
The list of "source" datacenters and "target" datacenters can be pointed in the `KmmConfig` Custom Resource too. 

# Installation 

To turn on KMM Configuration mode, the `operator.kmmConfiguratorEnabled` Helm property is used.
"Turn on this mode" depicts that an appropriate Custom Resource Definition is installed on an environment and Kafka Service Operator
watching namespaces are defined using the `operator.watchNamespace` Helm property. To watch several namespaces, Kafka Service Operator
should use Cluster Role instead of Role. In restricted environment, you need to create the Cluster Role manually.
To do this, see [Deployment Permissions](installation.md#deployment-permissions).
When the Kafka Service Operator watches several namespaces it affects only `KmmConfig` Custom Resources.
The `KafkaService` Custom Resources from other namespaces are not processed.

# KmmConfig Custom Resource Processing

The following is an example of `KmmConfig` Custom Resource:

```yaml
apiVersion: qubership.org/v1
kind: KmmConfig
metadata:
  name: example-kmm-config
  namespace: kafka-service
spec:
  kmmTopics:
    topics: topic1,topic2
    sourceDc: dc2
    targetDc: dc1
    transformation:
      transforms:
        - name: HeartbeatHeaderFilter
          type: com.qubership.kafka.mirror.extension.HeaderFilter
          params:
            filter.type: exclude
            headers: heartbeat,type=heartbeat
            topics: ${replication_prefix}topic1,${replication_prefix}topic2
        - name: PingHeaderFilter
          predicate: HasPingHeaderKey
          type: org.apache.kafka.connect.transforms.Filter
      predicates:
        - name: HasPingHeaderKey
          type: org.apache.kafka.connect.transforms.predicates.HasHeaderKey
          params:
            name: ping
```

Where:

* `topics` is a comma separated list of topic names or regexp for topic names. These topic names merge with already existing replication
  topic names.
* `sourceDc` is a comma separated list of "source" datacenters. Kafka "source datacenter" is the Kafka cluster from which Kafka Mirror Maker
  replicates the topics. The pointed datacenters must be contained in the `clusters` Kafka Mirror Maker Config Map property.
  If some of the datacenters do not satisfy this condition, the current Custom Resource processes with errors in the CR status and
  the topics are not added to appropriate list.
* `targetDc` is a comma separated list of "target" datacenters. Kafka "target datacenter" is the Kafka cluster to
   which Kafka Mirror Maker replicates the topics. The pointed datacenters must be either contained in the `target.dc`
   Kafka Mirror Maker Config Map property or contained in the `clusters` property if `target.dc` property does not exist.
* `transformation.transforms` is a list of transformations to make lightweight message-at-a-time modifications.
  For more detailed information, refer to [Transformations](https://kafka.apache.org/documentation/#connect_transforms).
  Each transformation has the following parameters:
  * `name` - Alias for the transformation. This parameter is mandatory.
  * `type` - Fully qualified class name for the transformation. This parameter is mandatory.
  * `predicate` - Alias of predicate to be associated with the transformation.
  * `negate` - Parameter allows to invert associated condition of associated predicate.
  * `params` - List of configuration properties for the transformation.
    Parameter value can contain `${replication_prefix}` placeholder which can be replaced with the following:
    - prefix in format `<sourceClusterName>.` (for example: `dc2.`) if `mirrorMaker.replicationPrefixEnabled` is set to `true` and 
      `global.disasterRecovery.mirrorMakerReplication.enabled` is set to `false` in [installation parameters](installation.md#parameters).
    - empty string if `mirrorMaker.replicationPrefixEnabled` is set to `false` or 
      `global.disasterRecovery.mirrorMakerReplication.enabled` is set to `true` in [installation parameters](installation.md#parameters).
* `transformation.predicates` is a list of predicates to be applied to some transformations.
  For more detailed information, refer to [Predicates](https://kafka.apache.org/documentation/#connect_predicates).
  Each predicate has the following parameters:
    * `name` - Alias for the predicate. This parameter is mandatory.
    * `type` - Fully qualified class name for the predicate. This parameter is mandatory.
    * `params` - List of configuration properties for the predicate. Parameter value can contain `${replication_prefix}`
      placeholder which can be replaced with the following:
      - prefix in format `<source datacenter>.` if `mirrorMaker.replicationPrefixEnabled` is set to `true` and
        `global.disasterRecovery.mirrorMakerReplication.enabled` is set to `false` in [installation parameters](installation.md#parameters).
      - empty string if `mirrorMaker.replicationPrefixEnabled` is set to `false` or `global.disasterRecovery.mirrorMakerReplication.enabled`
        is set to `true` in [installation parameters](installation.md#parameters).

**Pay attention**, custom transformations are applied to all messages produced by Kafka Mirror Maker, including system ones such
as heartbeats and checkpoints, so use predicates or transformations that allows to limit the topics to which the transformation is applied.

Kafka Service Operator processes all caught `KmmConfig` Custom Resources. It uses the following algorithm:

1. Checks if Kafka Service Operator allows the usage of the "source" and "target" datacenters in the current CR.
2. Creates all possible pairs of source -> target datacenters.
3. For each "source -> target" pair sets the `enabled` replication property to `true`, adds transformation properties and creates 
   or merges with the existing `topics` list.

For example, you have the following part of KMM Config Map:

```text
clusters = dc2, dc1
...
target.dc = dc1
...
dc2->dc1.enabled = false
dc2->dc1.topics = topic3
...
```

After processing above `KmmConfig` CR, the KMM Config Map changes to the following:

```text
clusters = dc2, dc1
...
target.dc = dc1
...
dc2->dc1.enabled = true
dc2->dc1.transforms.example-kmm-config_kafka-service_HeartbeatHeaderFilter.type = com.qubership.kafka.mirror.extension.HeaderFilter
dc2->dc1.transforms.example-kmm-config_kafka-service_HeartbeatHeaderFilter.filter.type = exclude
dc2->dc1.transforms.example-kmm-config_kafka-service_HeartbeatHeaderFilter.headers = heartbeat,type=heartbeat
dc2->dc1.transforms.example-kmm-config_kafka-service_HeartbeatHeaderFilter.topics = dc2.topic1,dc2.topic2
dc2->dc1.transforms.example-kmm-config_kafka-service_PingHeaderFilter.type = org.apache.kafka.connect.transforms.Filter
dc2->dc1.transforms.example-kmm-config_kafka-service_PingHeaderFilter.predicate = example-kmm-config_kafka-service_HasPingHeaderKey
dc2->dc1.predicates.example-kmm-config_kafka-service_HasPingHeaderKey.type = org.apache.kafka.connect.transforms.predicates.HasHeaderKey
dc2->dc1.predicates.example-kmm-config_kafka-service_HasPingHeaderKey.name = ping
dc2->dc1.transforms = example-kmm-config_kafka-service_HeartbeatHeaderFilter,example-kmm-config_kafka-service_PingHeaderFilter
dc2->dc1.predicates = example-kmm-config_kafka-service_HasPingHeaderKey
dc2->dc1.topics = topic3,topic1,topic2
...
```

Kafka Service Operator tracks KMM Config Map and reboots Kafka Mirror Maker after all Config Map changes.

# Troubleshooting 

Each `KmmConfig` Custom Resource changes status after processing. The following are two status properties for `KmmConfig` Custom Resource:

* `IsProcessed` property is `true`, if the CR has already been processed by Kafka Service Operator.
* `problemTopics` property contains messages with all occurred processing errors as plain text.
