# Prometheus Alerts

## KafkaIsDegradedAlert

### Description

Kafka cluster degraded, it means that at least one of the nodes have failed, but cluster is able to work.

For more information refer to [Kafka is Degraded](./troubleshooting.md#kafka-is-degraded).

### Possible Causes

- Kafka pod failures or unavailability.
- Resource constraints impacting Kafka pod performance.

### Impact

- Reduced or disrupted functionality of the Kafka cluster.
- Potential impact on processes relying on the Kafka.

### Actions for Investigation

1. Check the status of Kafka pods.
2. Review logs for Kafka pods for any errors or issues.
3. Verify resource utilization of Kafka pods (CPU, memory).

### Recommended Actions to Resolve Issue

1. Restart or redeploy Kafka pods if they are in a failed state.
2. Investigate and address any resource constraints affecting the Kafka pod performance.

## KafkaMetricsAreAbsent

### Description

Kafka monitoring metrics are absent.

### Possible Causes

- Monitoring is not properly configured.
- Network issues between kafka and Prometheus.

### Impact

- Absense of Kafka metrics.

### Actions for Investigation

1. Check the monitoring configuration.

### Recommended Actions to Resolve Issue

1. Check if monitoring configuration is correct and redeploy Kafka pods.

## KafkaIsDownAlert

### Description

Kafka cluster is down, and there are no available pods.

For more information refer to [Kafka is Down](./troubleshooting.md#kafka-is-down).

### Possible Causes

- Network issues affecting the Kafka pod communication.
- Kafka's storage is corrupted.
- Internal error blocks Kafka cluster working.

### Impact

- Complete unavailability of the Kafka cluster.
- Other processes relying on the Kafka cluster will fail.

### Actions for Investigation

1. Check the status of Kafka pods.
2. Review logs for Kafka pods for any errors or issues.

### Recommended Actions to Resolve Issue

1. Restart or redeploy Kafka pods if they are in a failed state.
2. Investigate and address any resource constraints affecting the Kafka pod performance.

## KafkaCPUUsageAlert

### Description

One of Kafka pods CPU consumption reaches the resource limit.

For more information refer to [CPU Limit](./troubleshooting.md#cpu-limit-reached).

### Possible Causes

- Insufficient CPU resources allocated to Kafka pods.
- The service is overloaded.
- [Kafka CPU is overloaded only for one of the cluster nodes](./troubleshooting.md#kafka-cpu-is-overloaded-only-for-one-of-the-cluster-nodes)

### Impact

- Increased response time and potential slowdown of Kafka requests.
- Degraded performance of services used the Kafka.
- Potential Kafka failure when CPU consumption reaches resource limit for particular Kafka.

### Actions for Investigation

1. Monitor the CPU consumption trends in Kafka Monitoring dashboard.
2. Review Kafka logs for any performance related issues.

### Recommended Actions to Resolve Issue

1. Try to increase CPU request and CPU limit for Kafka.
2. Scale up Kafka cluster as needed.
3. Perform rebalance of Kafka cluster.

## KafkaCPULoadAlert

### Description

One of Kafka pods came close to the CPU limit.

For more information refer to [CPU Limit](./troubleshooting.md#cpu-limit-reached).

### Possible Causes

- Insufficient CPU resources allocated to Kafka pods.
- The service is overloaded.

### Impact

- Increased response time and potential slowdown of Kafka requests.
- Degraded performance of services used the Kafka.
- Potential Kafka failure when CPU consumption reaches resource limit for particular Kafka.

### Actions for Investigation

1. Monitor the CPU usage trends in Kafka Monitoring dashboard.
2. Review Kafka logs for any performance related issues.

### Recommended Actions to Resolve Issue

1. Try to increase CPU request and CPU limit for Kafka.
2. Scale up Kafka cluster as needed.

## KafkaMemoryUsageAlert

### Description

One of Kafka pods came close to the specified memory limit.

For more information refer to [Memory Limit](./troubleshooting.md#memory-limit-reached).

### Possible Causes

- Insufficient memory resources allocated to Kafka pods.
- Service is overloaded.

### Impact

- Potentially lead to the increase of response times or crashes.
- Degraded performance of services used the Kafka.

### Actions for Investigation

1. Monitor the Memory usage trends in Kafka Monitoring dashboard.
2. Review Kafka logs for memory related errors.

### Recommended Actions to Resolve Issue

1. Try to increase Memory request and Memory limit for Kafka.
2. Scale up Kafka cluster as needed.

## KafkaHeapMemoryUsageAlert

### Description

Heap memory usage by one of the pods in the Kafka cluster came close to the specified memory limit.

For more information refer to [Memory Limit](./troubleshooting.md#memory-limit-reached).

### Possible Causes

- Insufficient memory resources allocated to Kafka pods.
- Service is overloaded.

### Impact

- Potentially lead to the increase of response times or crashes.
- Degraded performance of services used the Kafka.

### Actions for Investigation

1. Monitor the Memory usage trends in Kafka Monitoring dashboard.
2. Review Kafka logs for memory related errors.
3. Verify resource utilization of Kafka pods (CPU, memory).

### Recommended Actions to Resolve Issue

1. Try to increase Heap Size for Kafka.
2. Scale up Kafka cluster as needed.

## KafkaGCCountAlert

### Description

Garbage collections count rate of one of the pods in the Kafka cluster comes close to the specified limit.

This limit can be overridden with parameter `thresholds.gcCountAlert` described in
[Kafka Monitoring Parameters](/docs/public/installation.md#monitoring).

For more information refer to [Memory Limit Reached](./troubleshooting.md#memory-limit-reached).

### Possible Causes

- Insufficient memory resources allocated to Kafka pods.
- Service is overloaded.

### Impact

- Potentially lead to the increase of response times or crashes.
- Degraded performance of services used the Kafka.

### Actions for Investigation

1. Monitor the Memory usage trends Kafka Monitoring dashboard.
2. Review Kafka logs for memory related errors.
3. Verify resource utilization of Kafka pods (CPU, memory).

### Recommended Actions to Resolve Issue

1. Try to increase Memory request, Memory limit and Heap Size for Kafka.
2. Scale up Kafka cluster if needed.

## KafkaLagAlert

### Description

The maximum consumer group offset lag across the Kafka cluster exceeds the configured threshold.

The threshold is configured with `monitoring.thresholds.lagAlert` (default `100000`). Set the parameter to `-1` to disable this alert. Requires Kafka Exporter (`monitoring.lagExporter.enabled: true`).

For more information refer to [Lag Limit Reached](./troubleshooting.md#lag-limit-reached).

### Possible Causes

- Consumer service is overloaded or stopped.
- Consumer group cannot keep up with the produce rate.

### Impact

- The Kafka data can be lost because its persistence is based on retention.

### Actions for Investigation

1. Monitor consumer group lag in the Kafka Monitoring or Kafka Exporter dashboard.
2. Check consumer pod health and logs.

### Recommended Actions to Resolve Issue

1. Consider the possibility of increasing the number of topic partitions.
2. Increase the number of consumers.

## KafkaEstimatedLagSecondsAlert

### Description

The estimated time (in seconds) required for a consumer group to catch up exceeds the configured threshold. The estimate is calculated as total offset lag divided by the recent produce rate (same formula as the Kafka Exporter dashboard).

The threshold is configured with `monitoring.thresholds.lagAlertSeconds` (default `3600`). Set the parameter to `-1` to disable this alert. Requires Kafka Exporter (`monitoring.lagExporter.enabled: true`).

For more information refer to [Lag Limit Reached](./troubleshooting.md#lag-limit-reached).

### Possible Causes

- Consumer service is overloaded or stopped.
- Produce rate is high relative to consumer throughput.

### Impact

- Growing lag increases the risk of data loss when topic retention is reached.

### Actions for Investigation

1. Compare offset lag and produce rate in the Kafka Exporter dashboard.
2. Check whether consumers are running and processing messages.

### Recommended Actions to Resolve Issue

1. Scale up consumers or increase partition count.
2. Investigate slow message processing in the consumer application.

## Custom Consumer Lag Alerts

### Description

Additional lag alerts can be defined in `monitoring.thresholds.lagAlertConfiguration`. Each entry creates Prometheus alerts filtered by topic and consumer group regular expressions. Alert names are taken from the mandatory `alertName` field.

When both offset lag (`lagAlert`) and estimated lag seconds (`lagAlertSeconds`) are enabled for the same entry, the seconds alert is named `<alertName>Seconds`.

Configuration is described in [Consumer Lag Alerts](/docs/public/installation.md#consumer-lag-alerts).

### Possible Causes

Same as [KafkaLagAlert](#kafkalagalert) and [KafkaEstimatedLagSecondsAlert](#kafkaestimatedlagsecondsalert), but scoped to the configured topics and consumer groups.

### Impact

Same as [KafkaLagAlert](#kafkalagalert).

### Actions for Investigation

1. Identify the firing alert name and check the topic/group filters in the alert description.
2. Monitor the affected consumer groups in the Kafka Exporter dashboard.

### Recommended Actions to Resolve Issue

Same as [KafkaLagAlert](#kafkalagalert).

## KafkaMirrorMakerIsDegradedAlarm

### Description

At least one of the Kafka Mirror Maker nodes have failed.

For more information refer to [Kafka Mirror Maker is Degraded](./troubleshooting.md#kafka-mirror-maker-is-degraded).

### Possible Causes

- Left or right part of Disaster Recovery schema has failed.

### Impact

- DR can't be used properly, since one of the sides is `down` or `degraded`.

### Actions for Investigation

1. Check both left and right part of Disaster Recovery schema.

### Recommended Actions to Resolve Issue

1. Try to up Kafka Service and reboot appropriate Kafka Mirror Maker.

## KafkaMirrorMakerIsDownAlarm

### Description

All the Kafka Mirror Maker nodes have failed.

For more information refer to [Kafka Mirror Maker is Down](./troubleshooting.md#kafka-mirror-maker-is-down).

### Possible Causes

- Left and right part of Disaster Recovery schema have failed.

### Impact

- DR can't be used properly, since both sides have `failed` status.

### Actions for Investigation

1. Check both left and right part of Disaster Recovery schema.

### Recommended Actions to Resolve Issue

1. Try to up all Kafka Services and reboot Kafka Mirror Maker pods.

## KafkaPartitionCountAlert

### Description

Partition count of one of the broker in the Kafka cluster comes close to the specified limit.
There are strong restrictions for every Kafka cluster type and allowed number of partitions,
you can find then in [HWE](/docs/public/installation.md#hwe).

This limit can be overridden with parameter `thresholds.partitionCountAlert` described in [Kafka Monitoring Parameters](/docs/public/installation.md#monitoring)

### Possible Causes

- One of the Kafka pods or whole Kafka cluster is overloaded.

### Impact

- Impacts Kafka pod performance.

### Actions for Investigation

1. Monitor partition count of Kafka pods in Kafka Monitoring dashboard or Cruise Control.

### Recommended Actions to Resolve Issue

1. Perform rebalance of Kafka cluster if only one pod has overloaded partition number.
   You can find the rebalance command in [Topics with Insufficient Replication Factor](./troubleshooting.md#topics-with-insufficient-replication-factor).
2. Refer [Kafka Works Slowly or Consumes a lot of CPU For All Nodes](./troubleshooting.md#kafka-works-slowly-and-consumes-a-lot-of-cpu-for-all-nodes)
   to see options when the number or partitions are exceeded for whole cluster.

## KafkaBrokerSkewAlert

### Description

Partitions skew of one of the broker in the Kafka cluster comes close to the specified limit.

This limit can be overridden with parameter `thresholds.brokerSkewAlert` described in [Kafka Monitoring Parameters](/docs/public/installation.md#monitoring)

### Possible Causes

- One of the Kafka pods is overloaded.

### Impact

- Impacts Kafka pod performance.

### Actions for Investigation

1. Monitor broker skew of one of the Kafka pod in Kafka Monitoring dashboard.

### Recommended Actions to Resolve Issue

1. Perform rebalance of Kafka cluster.

## KafkaBrokerLeaderSkewAlert

### Description

Partitions skew of one of the broker in the Kafka cluster comes close to the specified limit.

This limit can be overridden with parameter `thresholds.brokerSkewAlert` described in [Kafka Monitoring Parameters](/docs/public/installation.md#monitoring)

### Possible Causes

- One of the Kafka pods is overloaded.

### Impact

- Impacts Kafka pod performance.

### Actions for Investigation

1. Monitor partitions skew of one of the broker in Kafka Monitoring dashboard.

### Recommended Actions to Resolve Issue

1. Perform rebalance of Kafka cluster.

## SupplementaryServicesCompatibilityAlert

### Description

Kafka supplementary services in namespace is not compatible with installed Apache Kafka version,
allowed range of supported version is provided by supplementary services.

for more information refer to [Upgrade Guide](/docs/public/installation.md#upgrade)

### Possible Causes

- Supplementary services version is not compatible with Apache Kafka version.

### Impact

- Supplementary services won't be able to work with Kafka.
- Other processes relying on the Kafka cluster will fail.

### Actions for Investigation

1. Check compatibility of services.

### Recommended Actions to Resolve Issue

1. Install compatible versions of Kafka supplementary services and Apache Kafka version.
