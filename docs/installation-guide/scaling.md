The section provides information about Kafka Cluster Scaling.

# Prerequisites

Ensure you are using the appropriate CRD version, v.2.6.1-3.10 version or newer.
It must contain `scaling` and `partitionsReassignmentStatus` parameters.
If currently used CRD is older, apply new CRD. For more information, see 
[Custom Resource Definition with Versioning](installation.md#custom-resource-definition-with-versioning).

# Scaling

To scale out Kafka cluster:

1. Specify the value for `kafka.replicas` with desired increased number of brokers.
2. If you are using `kafka.racks`, add racks for new brokers.
3. If you are using `kafka.storage.volumes`, add volumes for new brokers.
4. If you are using `kafka.storage.labels`, add labels for new brokers.
5. If you are using `kafka.storage.nodes`, add node names for new brokers.
6. Optionally, you can set `kafka.scaling.reassignPartitions` to `true` to allow partitions be reassigned automatically among new brokers.

**Note**: It is recommended to reassign partitions among new brokers after scale in. For production and large environments 
(more than 4000 partitions) we recommend to use [Cruise Control rebalance operation](./cruise-control.md#rebalance-cluster-1) 
if you have it in your delivery or manual 
operation for rebalance topics, using the following command on any Kafka pod after start of new brokers:

```sh
./bin/kafka-partitions.sh rebalance_leaders
```

1. Optionally, change the `kafka.scaling.allBrokersStartTimeoutSeconds` and `kafka.scaling.topicReassignmentTimeoutSeconds` parameter values
   if default values are not suitable. For example, the values are too small.
2. Specify the `monitoring.kafkaTotalBrokerCount` to the new `kafka.replicas` if Kafka Monitoring is enabled.

New brokers are added to the cluster, default replication factor is set to 3, and all existing partitions are reassigned among all brokers.
If previous `kafka.replicas` value was less than 3, old brokers are rebooted after partitions reassignment to apply new default replication 
factor as 3.
