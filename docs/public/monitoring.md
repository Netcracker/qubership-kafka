This document describes monitoring dashboards, their metrics, Zabbix alarms and Prometheus alerts.

- [Overview](#overview)
- [Kafka Monitoring](#kafka-monitoring)
- [Kafka Lag Exporter](#kafka-lag-exporter)
- [Kafka Topics](#kafka-topics)
- [Kafka Mirror Maker Monitoring](#kafka-mirror-maker-monitoring)
- [Table of Metrics](#table-of-metrics)
- [Monitoring Alarms Description](#monitoring-alarms-description)

# Overview

The dashboards provide the following parameters to configure at the top of the dashboard:

* Interval time for metric display
* Node name

For all graph panels, the mean metric value is used in the given interval. For all `singlestat` panels, the last metric value is used.

# Kafka Monitoring

This section describes the Kafka Monitoring dashboard and its metrics.

## Dashboard

Some widgets on the **Cluster Overview** panel are different for InfluxDB and Prometheus.
Prometheus dashboard shows information about ready pods, but InfluxDB dashboard does not contain information about ready pods.

An overview of Kafka Monitoring dashboard for Prometheus is shown below.

![Prometheus Dashboard](/docs/public/images/kafka-monitoring_dashboard-prometheus.png)

An overview of Kafka Monitoring dashboard for InfluxDB is shown below.

![InfluxDB Dashboard](/docs/public/images/kafka-monitoring_dashboard-influxdb.png)

### Metrics

**Cluster Overview**

Prometheus panel:

![Cluster Overview Prometheus](/docs/public/images/kafka-monitoring_cluster_overview-prometheus.png)

InfluxDB panel:

![Cluster Overview InfluxDB](/docs/public/images/kafka-monitoring_cluster_overview-influxdb.png)

* `Cluster status` - Displays the status of Kafka cluster.
* `Current cluster size` - Displays the current size of Kafka cluster.
* `Ready Pods` - Displays the count of the ready pods.

   **Note**: This widget is displayed only for the Prometheus dashboard. Not applicable for external managed Kafka (e.g. Amazon MSK).

* `Current controller broker id` - Displays the broker ID of the current controller. The first node to boot in 
  a Kafka cluster automatically becomes the controller, and there can be only one. The controller 
  in a Kafka cluster is responsible for maintaining the list of partition leaders and coordinating 
  leadership transitions (in the event a partition leader becomes unavailable). If it becomes 
  necessary to replace the controller, a new controller is randomly chosen by ZooKeeper from the pool 
  of brokers. In general, it is not possible for this value to be greater than one, but you should 
  definitely get an alert on a value of zero.
* `Controller State` - Displays the state of the controller broker. The controller changes state when processing an event. 
  Most of the time, the `Idle` state is displayed, which depicts that the controller has processed all the events. 
  If the event processing takes a considerable amount of time, several seconds or more, the state matching the processing event is
  displayed.
  The controller can have one of the following states:
  * Idle
  * ControllerChange
  * BrokerChange
  * TopicChange
  * TopicDeletion
  * AlterPartitionReassignment
  * AutoLeaderBalance
  * ManualLeaderBalance
  * ControlledShutdown
  * IsrChange
  * LeaderAndIsrResponseReceived
  * LogDirChange
  * ControllerShutdown
  * UncleanLeaderElectionEnable
  * TopicUncleanLeaderElectionEnable
  * ListPartitionReassignment
  * UpdateMetadataResponseReceived
  * KRaft
* `Kafka version` - Displays the Kafka version running on the controller broker.
* `Similar Configs` - Displays the information about brokers configuration consistency. It displays `Yes` if the configuration
  is same across all brokers. Otherwise, it displays
  `No: [{name of any discrepant config}: {value on one broker} VS {different value on another broker}]`.
* `Pods Readiness Probe` - Displays the readiness probe state for each Kafka pod.

  **Note**: This widget is displayed only for Prometheus dashboard. Not applicable for external managed Kafka (e.g. Amazon MSK).

* `Broker Metrics` - Displays the information about brokers as a table:
   * Broker ID - The ID of the broker.
   * Broker State - The broker can have one of the following states:
     * NotRunning
     * Starting
     * RecoveringFromUncleanShutdown
     * RunningAsBroker
     * PendingControlledShutdown
     * BrokerShuttingDown
   * Partitions - The number of partition replicas on the broker.
   * Broker Skew - The deviation in percent of the average number of partitions per broker. It can be negative if the partitions number
     on the broker is less than the average.
   * Partition Leadership - The number of partitions where the broker is a leader.
   * Broker Leader Skew - The deviation in percent of the average number of partition leadership per broker. 
     It can be negative if the broker is a leader for fewer number of partitions than the average.
   * Under Replicated Partitions - The number of under-replicated partitions (|ISR| < |current replicas|).
   * Under Min ISR Partitions - The number of partitions whose in-sync replicas count is less than Minimum In-sync Replicas.

**Broker Issues**

![Broker Issues](/docs/public/images/kafka-monitoring_broker_issues.png)

* `Under Replicated Partitions` - The number of under-replicated partitions. Replicas that are added as part of
   a reassignment will not count toward this value.
* `Under Min ISR Partitions` - The number of partitions whose in-sync replicas (ISR) count is less than `minIsr`.
* `Offline Partitions` - The number of partitions that don’t have an active leader and are hence not writable or readable.
* `Partitions With Late Transactions` - The number of partitions which have open transactions with durations exceeding
  `transaction.max.timeout.ms` (plus 5 minutes).
* `Broker Errors Rate` - The rate of errors in responses counted per error code. If a response contains multiple errors, all are counted.

**Topics**

![Topics](/docs/public/images/kafka-monitoring_topics.png)

* `Total Topics Count` - Displays the total count of topics on Kafka cluster.
* `Total Partitions Count` - Displays the total count of partitions on Kafka cluster.
* `Max Partitions Per Broker` - Displays the number of max partitions per broker. May be not applicable for managed external Kafka.
* `Topics Without Leader` - Displays the count of topics which have at least one partition without a leader (offline partitions).
* `Offline partitions count` - Displays the number of partitions without an active leader. All read and 
  write operations are only performed on partition leaders, a non-zero value for this metric 
  should be alerted to prevent service interruptions. Any partition without an active leader 
  is completely inaccessible, and both consumers and producers of that partition are blocked 
  until a leader becomes available.
  Not applicable for external managed Kafka (e.g. Amazon MSK).
* `Under Replicated Topics` - Displays the count of topics which have at least one under-replicated partition.
  Not applicable for external managed Kafka (e.g. Amazon MSK).
* `Topics With Unclean Leader Election Enabled` - Displays the count of topics with unclean leader election enabled. 
  Unclean leader elections can lead to data loss, so you should check these topics to see if this setting is set reasonably.
  Not applicable for external managed Kafka (e.g. Amazon MSK).

**Consumer Groups Metrics**

![Consumer Groups Metrics](/docs/public/images/kafka-monitoring_consumer_groups_metrics.png)

* `Consumer Groups Number` - Displays the number of consumer groups in different states as a table.

**JVM Heap and GC Metrics**

![JVM Heap and GC Metrics](/docs/public/images/kafka-monitoring_jvm_heap_and_gc_metrics.png)

* `JVM Heap Usage` - Displays the JVM heap usage by broker.
* `JVM Heap Usage in Percent` - Displays the JVM heap usage by broker in percent (%).
* `GC Time` - Displays the garbage collection time rate per second by broker.
* `GC Count` - Displays the garbage collection count rate per second by broker.

NOTE: JVM metrics are not applicable for external managed Kafka (e.g. Amazon MSK).

**RAM Memory Metrics**

![RAM Memory Metrics](/docs/public/images/kafka-monitoring_ram_memory_metrics.png)

* `Memory usage` - Displays the memory usage by pod. Not applicable for external managed Kafka (e.g. Amazon MSK).

**CPU**

![CPU](/docs/public/images/kafka-monitoring_cpu.png)

* `CPU usage` - Displays the CPU usage by pod. Not applicable for external managed Kafka (e.g. Amazon MSK).

**Disk Metrics**

![Disk Metrics](/docs/public/images/kafka-monitoring_disk_metrics.png)

* `Disk Read Bytes` - Displays the total size in bytes of read operations on the volume for each broker.
* `Disk Write Bytes` - Displays the total size in bytes of write operations on the volume for each broker.
* `Topic Partition Data Size` - Displays the total size in bytes of volume space occupied by topic partitions logs for each broker.

**Controller Queues Metrics**

![Controller Queues Metrics](/docs/public/images/kafka-monitoring_controller_queues_metrics.png)

* `Total Queue Size` - Displays the total number of controller requests to be sent out to brokers by `ControllerChannelManager`. 
  `ControllerChannelManager` establishes connection to every broker and starts a corresponding `RequestSendThread` 
  to keep sending queued controller requests.
* `Event Queue Size` - Displays the size of the `ControllerEventManager`'s queue. 
  Every `ControllerEvent` has an associated state. When a `ControllerEvent` is processed, it triggers a state transition 
  to the requested state. The `ControllerEvent` events are managed by `ControllerEventManager`.

  **Note**: Normally values of `Total Queue Size` and `Event Queue Size` metrics should be small or zero, 
  because requests and events are sent quickly within milliseconds, and they do not stay in the queue for a long time. 
  The increase in the value of these metrics depicts that the Kafka controller and the whole cluster is under a heavy load.

* `Event Queue Time Rate` - Displays the time it takes for any event, except the `Idle` event, to wait 
  in the `ControllerEventManager`'s queue before being processed.

**Network Metrics**

![Network Metrics](/docs/public/images/kafka-monitoring_network_metrics.png)

* `Bytes In/Out` - Displays the aggregate incoming/outgoing byte rate per second.
* `Bytes Rejected` - Displays the amount of data in bytes per second rejected by broker.

**ZooKeeper Connection Metrics**

![ZooKeeper Connection Metrics](/docs/public/images/kafka-monitoring_zookeeper_connection_metrics.png)

* `ZooKeeper Session State` - Displays ZooKeeper session state for each broker as a table.
  Not applicable for external managed Kafka (e.g. Amazon MSK). The possible states are:
  * CONNECTING
  * ASSOCIATING
  * CONNECTED
  * CONNECTEDREADONLY
  * CLOSED
  * AUTH_FAILED
  * NOT_CONNECTED
* `ZooKeeper Requests Rate` - Rate of ZooKeeper requests. May be not applicable for managed external Kafka.
* `ZooKeeper Requests Latency` - Latency of ZooKeeper requests. May be not applicable for managed external Kafka.

**In Sync Replica (ISR) Metrics**

![ISR Metrics](/docs/public/images/kafka-monitoring_isr_metrics.png)

* `Isr Expands Rate` - Displays if a broker goes down, then the ISR for some partitions shrink.
  When that broker is up again, ISR is expanded after the replicas are fully caught up. 
  Other than that, the expected value for both ISR shrink and expansion rates is 0.
* `Isr Shrinks Rate` - Displays if a broker goes down, then ISR for some partitions shrink.
  When that broker is up again, ISR is expanded after the replicas are fully caught up. 
  Other than that, the expected value for both ISR shrink and expansion rates is 0.

**Messages Metric**

![Messages Metric](/docs/public/images/kafka-monitoring_messages_metric.png)

* `Message Rate` - Displays the aggregate incoming messages rate per second.

**Request Metrics**

![Request Metrics](/docs/public/images/kafka-monitoring_request_metrics.png)

* `Total Fetch Requests Rate` - Displays the total fetch request rate per second.
* `Total Produce Requests Rate` - Displays the total produce request rate per second.
* `Failed Fetch Requests Rate` - Displays the fetch request rate per second for requests that failed.
* `Failed Produce Requests Rate` - Displays the produce request rate per second for requests that failed.
* `Fetch Queue Size` - Displays the size of the fetch queue.
* `Request Queue Size` - Displays the size of the request queue. A congested request queue is not able to 
  process incoming or outgoing requests.

**Threads**

![Threads](/docs/public/images/kafka-monitoring_threads.png)

* `Thread count` - Displays the current number of live threads, including both daemon and non-daemon threads.
* `Total Started Thread Count` - Displays the total number of threads created and also started since the JVM started.
* `Log Cleaner And Replica Fetcher Threads` - Displays the information about configured and dead Log Cleaner and Replica Fetcher
  threads as a table.

**NOTE:** Kafka Monitoring dashboard has a link to `Node Details` dashboard which can be used for monitoring resources (CPU/Disk)
of external managed Kafka brokers (e.g. [Amazon MSK](managed/amazon.md#preparations-for-monitoring)).
Pay attention, this dashboard does not work with platform Kafka, and it could have a lot of empty panels for external Kafka.

# Kafka Lag Exporter

This section describes the Kafka Lag Exporter dashboard and their metrics.

## Dashboard

An overview of Kafka Lag Exporter dashboard is shown below.

![Dashboard](/docs/public/images/overview_kafka_lag_exporter.png)

**Note**: This dashboard is only available in Prometheus.

### Metrics

**All Consumer Group Lag**

![All Consumer Group Lag](/docs/public/images/all_consumer_group_lag.png)

* `Consumer Group Max Lag Seconds` - shows max extrapolated lag in seconds for each consumer group.
* `Consumer Group Lag Partition Seconds` - shows extrapolated lag in seconds for each partition.
* `Consumer Group Max Lag Offsets` - shows max offset lag for each consumer group.
* `Consumer Group Lag Partition Offsets` - shows offset lag for each consumer group per topic partition.

**Consumer Group Lag In Time Per Group Over Offset Lag**

![Consumer Group Lag In Time Per Group Over Offset Lag](/docs/public/images/Consumer_Group_Lag_In_Time_Per_Group_Over_Offset_Lag.png)

This panel describes the consumer groups individually. Each monitored Kafka consumer group has
own Grafana widget.

* `Consumer Group Lag In Time Per Group Over Offset Lag` - shows the max lag in time on the 
  left Y axis and max lag in offsets on the right Y axis for the current consumer group.

**Consumer Group Lag in Time Per Group Over Summed Offsets**

![Consumer Group Lag in Time Per Group Over Summed Offsets](/docs/public/images/Consumer_Group_Lag_in_Time_Per_Group_Over_Summed_Offsets.png)

This panel describes the consumer groups individually. Each monitored Kafka consumer group has
own Grafana widget.

* `Consumer Group Lag in Time Per Group Over Summed Offsets` - shows the max lag in time on the left Y axis. 
  The right Y axis has the sum of latest and last consumed offsets for all the current group partitions.

**Kafka Lag Exporter JVM Metrics**

![Kafka Lag Exporter JVM Metrics](/docs/public/images/Kafka_Lag_Exporter_JVM_Metrics.png)

* `JVM Memory Used` - displays JVM memory usage for Kafka Lag Exporter.
* `JVM GC Time` - displays JVM garbage collection time for Kafka Lag Exporter.
* `JVM GC Rate` - displays JVM garbage collection rate for Kafka Lag Exporter.

# Kafka Topics

This section describes the `Kafka Topics` dashboard and their metrics.

## Dashboard

An overview of `Kafka Topics` dashboard is shown below.

![Dashboard](/docs/public/images/kafka-topics_dashboard.png)

**Note**: This dashboard is only available in Prometheus.

### Metrics

**Overview**

![Overview](/docs/public/images/kafka-topics_overview.png)

* `Topics` - The count of partitions, number of messages and size in bytes for each topic in descending order of size values presented
  as a table.

**Topic Issues**

![Topic Issues](/docs/public/images/kafka-topics_topic_issues.png)

* `Under Min ISR Partitions Table` - The list of topic partitions whose in-sync replicas (ISR) count is less than `minIsr` for each broker.
* `Under Replicated Partitions Table` - The list of topic under-replicated partitions for each broker.
* `Unclean Election Leader Topics Table` - The list of topics with unclean leader election enabled. Unclean leader elections can lead to
 data loss, so you should check these topics to see if this setting is set reasonably. May be not applicable for managed external Kafka.
* `Under Replicated Partitions` - The number of topic under-replicated partitions for each broker.
* `Under Min ISR Partitions` - The number of topic partitions whose in-sync replicas (ISR) count is less than `minIsr` for each broker.

**Topics Information**

![Topics Information 1](/docs/public/images/kafka-topics_topics_information_1.png)

![Topics Information 2](/docs/public/images/kafka-topics_topics_information_2.png)

* `Incoming Messages Rate` - The incoming messages rate by specific topic for each broker. `No Data` for specific topic means that
  there are no operations performed on the topic.
* `Incoming Bytes Rate` - The incoming bytes rate by specific topic for each broker. `No Data` for specific topic means that
  there are no operations performed on the topic.
* `Outgoing Bytes Rate` - The outgoing bytes rate by specific topic for each broker. `No Data` for specific topic means that
  there are no operations performed on the topic.
* `Partition Data Size` - The total size in bytes of volume space occupied by partitions logs of specific topic for each broker.

# Kafka Mirror Maker Monitoring

This section describes the Kafka Mirror Maker Monitoring dashboard and their metrics.

## Dashboard

An overview of Kafka Mirror Maker Monitoring dashboard is shown below.

![Dashboard](/docs/public/images/kmm-monitoring_dashboard.jpg)

### Metrics

**Overall status**

![Overall status](/docs/public/images/kmm-monitoring_overall_status.jpg)

* `KMM Status` - shows current state of kafka-mirror-maker. It can be UP, DEGRADED, FAILED.
* `Node Status` - shows the amount of alive and failed instances.

**JVM overview**

![Heap Memory Usage](/docs/public/images/kmm-monitoring_heap_memory_usage.jpg)

* `Heap Memory Usage` - shows the amount of memory that is using for the heap.

![Eden Space](/docs/public/images/kmm-monitoring_eden_space.jpg)

* `Eden Space` - shows the amount of memory that is using for eden space.

![Survivor Space](/docs/public/images/kmm-monitoring_survivor_space.jpg)

* `Survivor Space` - shows the amount of memory that is using for survivor space.

![Non-Heap Memory Usage](/docs/public/images/kmm-monitoring_non_heap_memory_usage.jpg)

* `Non-Heap Memory Usage` - shows the amount of memory that is using for non-heap area.

![Metaspace](/docs/public/images/kmm-monitoring_metaspace.jpg)

* `Metaspace` - shows the amount of memory that is using for metaspace.

![Thread Count](/docs/public/images/kmm-monitoring_thread_count.jpg)

* `Thread Count` - shows the amount of active threads.

**Replication status**

![Replication status](/docs/public/images/kmm-monitoring_replication_status.jpg)

* `Replicated bytes` - shows the amount of replicated bytes.
* `Replicated records` - shows the amount of replicated records.
* `Record age` - shows age of each record when consumed.
* `Replication speed per second` - shows the amount of replicated bytes per second.
* `Replication latency` - shows a timespan between each record's timestamp and downstream ACK.

# Table of Metrics

This table provides full list of Prometheus metrics being collected by Kafka Monitoring.

| Metric name                                        | Description                                                                                                     | Prometheus name |  Amazon MSK   |  Aiven Kafka  | Qubership Kafka Service |
|----------------------------------------------------|:----------------------------------------------------------------------------------------------------------------|:---------------:|:-------------:|:-------------:|:------------------------:|
| kafka_cluster_status                               | Status of the Kafka cluster. Internal metric of monitoring agent, including labels for topics, partitions, etc. |                 | Not Supported |   Supported   |        Supported         |
| kafka_cluster_Partition_Value                      | Information about partition state                                                                               |                 |   Supported   |   Supported   |        Supported         |
| kafka_cluster_topic_without_leader                 | Information about topics withot leader                                                                          |                 |   Supported   |   Supported   |        Supported         |
| kafka_cluster_unclean_election_topics              | Information about topics with enabled unclean leader election                                                   |                 |   Supported   |   Supported   |        Supported         |
| kafka_cluster_quorum_mode                          | Describes whether Kafka deployed in KRaft mode or Kafka with ZooKeeper mode                                        |                 |   Supported   |   Supported   |        Supported         |
| kafka_controller_KafkaController_Value             | State of the controller broker. Including information about topics and partitions.                              |                 |   Supported   |   Supported   |        Supported         |
| kafka_server_app_info_Version                      | Kafka version running on controller broker                                                                      |                 | Not Supported |   Supported   |        Supported         |
| kafka_server_KafkaServer_Value                     | Information about broker states.                                                                                |                 |   Supported   |   Supported   |        Supported         |
| kafka_controller_ControllerStats_Count             | Unclean Leader Election Count                                                                                   |                 |   Supported   |   Supported   |        Supported         |
| kafka_coordinator_group_GroupMetadataManager_Value | Consumer groups metric                                                                                          |                 |   Supported   |   Supported   |        Supported         |
| java_Memory_HeapMemoryUsage_used                   | JVM heap usage by broker                                                                                        |                 | Not Supported |   Supported   |        Supported         |
| java_Memory_HeapMemoryUsage_max                    | JVM heap available by broker                                                                                    |                 | Not Supported |   Supported   |        Supported         |
| java_GarbageCollector_CollectionTime               | Garbage collection time                                                                                         |                 | Not Supported |   Supported   |        Supported         |
| java_GarbageCollector_CollectionCount              | Garbage collection count                                                                                        |                 | Not Supported |   Supported   |        Supported         |
| java_Threading_ThreadCount                         | Number of live threads                                                                                          |                 | Not Supported | Not Supported |        Supported         |
| java_Threading_TotalStartedThreadCount             | Number of started threads                                                                                       |                 | Not Supported | Not Supported |        Supported         |
| kafka_controller_ControllerChannelManager_Value    | Controller channel info                                                                                         |                 |   Supported   |   Supported   |        Supported         |
| kafka_controller_ControllerEventManager_Value      | Controller event info                                                                                           |                 |   Supported   |   Supported   |        Supported         |
| kafka_controller_ControllerEventManager_Count      | Time it takes for any event (except the Idle event) to wait in the ControllerEventManager's queue               |                 |   Supported   |   Supported   |        Supported         |
| kafka_consumergroup_group_lag                      | Consumer group lag                                                                                              |                 |   Supported   | Not Supported |        Supported         |
| kafka_server_BrokerTopicMetrics_Count              | Incoming/outgoing message or byte rate                                                                          |                 |   Supported   |   Supported   |        Supported         |
| kafka_server_SessionExpireListener_Value           | ZooKeeper session state for each broke                                                                          |                 | Not Supported | Not Supported |        Supported         |
| kafka_server_ZooKeeperClientMetrics_Count          | Latency of ZooKeeper requests                                                                                   |                 |   Supported   | Not Supported |        Supported         |
| kafka_server_ReplicaManager_Count                  | Count of replica manager                                                                                        |                 |   Supported   |   Supported   |        Supported         |
| kafka_server_Fetch_queue_size                      | Size of the fetch queue                                                                                         |                 |   Supported   | Not Supported |        Supported         |
| kafka_network_RequestChannel_Value                 | Size of the request queue                                                                                       |                 |   Supported   |   Supported   |        Supported         |
| kafka_network_RequestMetrics_Count                 | Information about requests to Kafka                                                                             |                 |   Supported   |   Supported   |        Supported         |
| kafka_log_cleaner_threads_count                    | Log Cleaner Threads Count                                                                                       |                 |   Supported   | Not Supported |        Supported         |
| kafka_log_Log_Value                                | Number of topic log segments, start/end offsets and size of topic logs                                          |                 |   Supported   | Not Supported |        Supported         |
| kafka_replica_fetcher_threads_count                | Replica Fetcher Threads Count                                                                                   |                 |   Supported   | Not Supported |        Supported         |
| service:tls_status:info                            | Shows the status of TLS for service.                                                                            |                 |   Supported   |   Supported   |        Supported         |
| kafka_server_ReplicaManager_Value                  | Information about replica's partitions                                                                          |                 |   Supported   |   Supported   |        Supported         |
| kafka_broker_skew                                  | Skew of broker partitions in percent                                                                            |                 |   Supported   |   Supported   |        Supported         |
| kafka_broker_leader_skew                           | Skew of broker leader partitions in percent                                                                     |                 |   Supported   |   Supported   |        Supported         |

# Monitoring Alarms Description

This section describes monitoring alarms. 

| Name                                    | Summary                                                                                                                                                                                                          | For | Severity | Expression Example                                                                                                                                                                                                                                          | Description                                                                                                                              | Troubleshooting Link                                                                                      | Amazon MSK    | Aiven Kafka   |
|-----------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-----|----------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------|-----------------------------------------------------------------------------------------------------------|---------------|---------------|
| KafkaIsDegradedAlert                    | Some of Kafka Service pods are down                                                                                                                                                                              | 3m  | high     | `kafka_cluster_status{container="kafka-monitoring",namespace="kafka-service"} == 6`                                                                                                                                                                         | Kafka is Degraded                                                                                                                        | [KafkaIsDegradedAlert](/docs/public/troubleshooting.md#kafka-is-degraded)                                 | Supported     | Supported     |
| KafkaMetricsAreAbsent                   | Kafka monitoring metrics are absent                                                                                                                                                                              | 3m  | high     | `absent(kafka_cluster_status{namespace="kafka-service"}) == 1`                                                                                                                                                                                              | Kafka metrics are absent on kafka-service                                                                                                | [KafkaMetricsAreAbsent](/docs/public/alerts.md#kafkametricsareabsent)                                     | Supported     | Supported     |
| KafkaIsDownAlert                        | All of Kafka Service pods are down                                                                                                                                                                               | 3m  | critical | `kafka_cluster_status{container="kafka-monitoring",namespace="kafka-service"} == 10`                                                                                                                                                                        | Kafka is Down                                                                                                                            | [KafkaIsDownAlert](/docs/public/troubleshooting.md#kafka-is-down)                                         | Supported     | Supported     |
| KafkaCPUUsageAlert                      | Some of Kafka Service pods load CPU higher then 95 percents                                                                                                                                                      | 3m  | high     | `max(rate(container_cpu_usage_seconds_total{container="kafka",namespace="kafka-service",pod=~"kafka-[0-9].*"}[5m])) / max(kube_pod_container_resource_limits_cpu_cores{exported_namespace="kafka-service",exported_pod=~"kafka-[0-9].*"}) > 0.95`           | Kafka CPU usage is higher than 95 percents                                                                                               | [KafkaCPUUsageAlert](/docs/public/troubleshooting.md#cpu-limit-reached)                                   | Not Supported | Not Supported |
| KafkaMemoryUsageAlert                   | Some of Kafka Service pods use memory higher then 95 percents                                                                                                                                                    | 3m  | high     | `max(container_memory_working_set_bytes{container="kafka",namespace="kafka-service",pod=~"kafka-[0-9].*"}) / max(kube_pod_container_resource_limits_memory_bytes{exported_namespace="kafka-service",exported_pod=~"kafka-[0-9].*"}) > 0.95`                 | Kafka memory usage is higher than 95 percents                                                                                            | [KafkaMemoryUsageAlert](/docs/public/troubleshooting.md#memory-limit-reached)                             | Not Supported | Not Supported |
| KafkaHeapMemoryUsageAlert               | Some of Kafka Service pods use heap memory higher then 95 percents                                                                                                                                               | 3m  | high     | `max(java_Memory_HeapMemoryUsage_used{broker=~"kafka-[0-9].*",namespace="kafka-service"}) / max(java_Memory_HeapMemoryUsage_max{broker=~"kafka-[0-9].*",namespace="kafka-service"}) > 0.95`                                                                 | Kafka heap memory usage is higher than 95 percents                                                                                       | [KafkaHeapMemoryUsageAlert](/docs/public/troubleshooting.md#memory-limit-reached)                         | Not Supported | Not Supported |
| KafkaGCCountAlert                       | Garbage collections count rate of one of the pods in the Kafka cluster higher than the specified limit                                                                                                           | 3m  | high     | `max(rate(java_GarbageCollector_CollectionCount{broker=~"kafka-[0-9].*",namespace="kafka-service"}[5m])) > limit`                                                                                                                                           | Some of Kafka Service pods have Garbage collections count rate higher than the limit                                                     | [KafkaGCCountAlert](/docs/public/troubleshooting.md#memory-limit-reached)                                 | Not Supported | Supported     |
| KafkaLagAlert                           | Some of Kafka Service pods have partition lag higher than specified limit                                                                                                                                        | 3m  | high     | `max(kafka_consumergroup_group_lag{namespace="kafka-service"}) > limit`                                                                                                                                                                                     | Some of Kafka Service pods have partition lag higher than specified limit                                                                | [KafkaLagAlert](/docs/public/troubleshooting.md#lag-limit-reached)                                        | Supported     | Not Supported |
| KafkaMirrorMakerIsDegradedAlarm         | At least one of the Kafka Mirror Maker nodes have failed                                                                                                                                                         | 3m  | high     | `kmm_health_status_code{project_name="{{ .Release.Namespace }}"} == 1`                                                                                                                                                                                      | Kafka Mirror Maker is Degraded                                                                                                           | [KafkaMirrorMakerIsDegradedAlarm](/docs/public/troubleshooting.md#kafka-mirror-maker-is-degraded)         | Not Supported |               |
| KafkaMirrorMakerIsDownAlarm             | All of Kafka Mirror Maker pods are down                                                                                                                                                                          | 3m  | high     | `kmm_health_status_code{project_name="{{ .Release.Namespace }}"} == 2`                                                                                                                                                                                      | Kafka Mirror Maker is Down                                                                                                               | [KafkaMirrorMakerIsDownAlarm](/docs/public/troubleshooting.md#kafka-mirror-maker-is-down)                 | Not Supported |               |
| KafkaPartitionCountAlert                | Some of Kafka Partition count is higher than the specified limit                                                                                                                                                 | 3m  | high     | `kafka_server_ReplicaManager_Value{broker=~"kafka-[0-9].*",name="PartitionCount",namespace="kafka-service"} > limit`                                                                                                                                        | Kafka Partition count for {{ $labels.broker }} broker is higher than specified limit                                                     | [KafkaPartitionCountAlert](/docs/public/alerts.md#kafkapartitioncountalert)                               | Supported     | Supported     |
| KafkaBrokerSkewAlert                    | Some of Kafka Broker Skew is higher than the specified limit                                                                                                                                                     | 3m  | high     | `kafka_broker_skew{broker=~"kafka-[0-9].*",container="kafka-monitoring",namespace="kafka-service"} > 50) and on (broker, namespace) (kafka_server_ReplicaManager_Value{broker=~"kafka-[0-9].*",name="PartitionCount",namespace="kafka-service"} > 3`        | Kafka Broker Skew for {{ $labels.broker }} broker is higher than the specified limit                                                     | [KafkaBrokerSkewAlert](/docs/public/alerts.md#kafkabrokerskewalert)                                       | Supported     | Supported     |
| KafkaBrokerLeaderSkewAlert              | Some of Kafka Broker Leader Skew is higher than the specified limit                                                                                                                                              | 3m  | high     | `kafka_broker_leader_skew{broker=~"kafka-[0-9].*",container="kafka-monitoring",namespace="kafka-service"} > 50) and on (broker, namespace) (kafka_server_ReplicaManager_Value{broker=~"kafka-[0-9].*",name="PartitionCount",namespace="kafka-service"} > 3` | Kafka Broker Leader Skew for {{ $labels.broker }} broker is higher than the specified limit                                              | [KafkaBrokerLeaderSkewAlert](/docs/public/alerts.md#kafkabrokerleaderskewalert)                           | Supported     | Supported     |
| SupplementaryServicesCompatibilityAlert | Kafka supplementary services in namespace {{ $labels.namespace }} is not compatible with Kafka version {{ $labels.application_version }}, allowed range is {{ $labels.min_version }} - {{ $labels.max_version }} | 3m  | high     | `supplementary_services_version_compatible{application="kafka",namespace="kafka-service"} != 1`                                                                                                                                                             | Kafka supplementary services in namespace {{ $labels.namespace }} is not compatible with Kafka version {{ $labels.application_version }} | [SupplementaryServicesCompatibilityAlert](/docs/public/alerts.md#supplementaryservicescompatibilityalert) | Supported     | Supported     |
