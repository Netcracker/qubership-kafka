---
name: troubleshoot-kafka
description: Diagnose qubership-kafka failures in Jira tickets, CI/CD pipelines, and project support. Use when a Kafka CR (netcracker.com/v1), the kafka or kafka-service chart, Kafka monitoring, Mirror Maker, or KRaft migration shows KafkaIsDegradedAlert, KafkaIsDownAlert, KafkaCPUUsageAlert, KafkaMemoryUsageAlert, KafkaHeapMemoryUsageAlert, KafkaLagAlert, KafkaMirrorMakerIsDegradedAlarm, KafkaMirrorMakerIsDownAlarm, No space left on device, Java heap space, ConnectionLoss, NoAuthException, ZooKeeperClientTimeoutException, runAsNonRoot, Permission denied on /var/opt/kafka/data, or node already exists.
---

# Troubleshoot Kafka

Fast entrypoint for Kafka assessment. Failure steps stay in `references/troubleshooting.md`. This file does not repeat them.

`references/troubleshooting.md` is the fact source for known failure modes. It is `docs/public/troubleshooting.md` from this repository, with the memory and disk procedures that guide points at appended as their own sections. Do not read it in full for a single symptom.

1. Grep headers with line numbers first: `grep -n '^#' references/troubleshooting.md`
2. Match the symptom against the jump table below, or the raw headers if the table is stale.
3. Read from the matched header through the next header of the same or higher level. A `##` section includes its `###` children; stop at the next `##` or `#`. Never load the whole file for one lookup.

Read one more section the same way only when the section already loaded points at it:

- Kafka Memory Limit → `## Memory limit procedure`
- Kafka Disk Filled on All Nodes → `## Disk filled procedure`

## Deployment

- Chart `operator/charts/helm/kafka` renders one `Kafka` (`netcracker.com/v1`) from `templates/cr.yaml`. Chart `operator/charts/helm/kafka-service` renders one `KafkaService` (`netcracker.com/v7`) plus monitoring, Mirror Maker, AKHQ, and the backup daemon.
- Broker defaults: `kafka.replicas` 3, data volume `kafka.storage.size` 1Gi mounted at `/var/opt/kafka/data`, client port 9092. `kafka.heapSize` defaults to 256. Memory request defaults to 600Mi and memory limit to 800Mi. `kafka.zookeeperConnect` defaults to `zookeeper:2181`. `kafka.kraft.enabled` defaults to false.
- Backup daemon is installed only when `backupDaemon.install` is true on the kafka-service chart; the chart default is false. Backup files live under `/opt/kafka/backup-storage`, not on the broker data volume.
- On a cluster that forces non-root: monitoring needs `securityContext.runAsUser: 1000`. KRaft migration controller volume writes need `securityContext.fsGroup: 1000` (`INFRA_KAFKA_FS_GROUP` when cloud integration is enabled). Do not hardcode `fsGroup` where the platform assigns the group, for example OpenShift SCC.

## Forbidden actions

- Do not `rm -rf` a partition directory under `/var/opt/kafka/data` unless the loaded section is Topics with Insufficient Replication Factor, after rebalance failed, or Kafka Unhealthy After Cluster Scale-In, for topics marked for deletion. That delete drops the partition data.
- Do not `rm -f` a segment `.log` unless the loaded section is the dirty-offset warning and the file is a zero-byte log named in that warning.
- Do not run `kafka-partition-logs.sh --delete` unless the loaded section is the under-replicated or heaviest partitions section, or Disk filled procedure. Do not use it on a compacted topic.
- Do not set `unclean.leader.election.enable=true` unless the loaded section is Kafka Unhealthy After Cluster Scale-In. Remove it after leaders are rebalanced.
- Do not `deleteall /admin/delete_topics` unless that same scale-in section already showed the topic there. Delete only the broken topics.
- Do not treat a full `/opt/kafka/backup-storage/` as a full broker data volume, or the reverse.

## Failure scenarios

Match one row, then read that section. No resolution steps live in this table.

| Symptom | Signal | Section header |
| --- | --- | --- |
| ZooKeeper used by Kafka is down | ClientCnxn closing the socket, failed ZooKeeper status | `## ZooKeeper Failure` |
| Partition leader or broker crashed | Cluster size dropped, `Node may not be available` | `## Partition Leader Crash` |
| Data volume full | `No space left on device` writing `/var/opt/kafka/data` | `## Data Is Out of Space` |
| At least one broker failed | `KafkaIsDegradedAlert` | `## Kafka is Degraded` |
| Every broker failed | `KafkaIsDownAlert` | `## Kafka is Down` |
| One Mirror Maker node failed | `KafkaMirrorMakerIsDegradedAlarm` | `## Kafka Mirror Maker is Degraded` |
| Every Mirror Maker node failed | `KafkaMirrorMakerIsDownAlarm` | `## Kafka Mirror Maker is Down` |
| CPU at the pod limit | `KafkaCPUUsageAlert` | `## CPU Limit Reached` |
| Memory at the pod limit | `KafkaMemoryUsageAlert`, `Java heap space` | `## Memory Limit Reached` |
| Consumer lag at the limit | `KafkaLagAlert`, `KafkaEstimatedLagSecondsAlert` | `## Lag Limit Reached` |
| Broker stuck connecting to ZooKeeper | `ZooKeeperClientTimeoutException`, state `CONNECTING` | `## Timed Out Waiting for ZooKeeper Connection` |
| Broker session to ZooKeeper dropped | `Client session timed out` | `## ConnectionLoss Error for ZooKeeper Connection` |
| Broker cannot write ZooKeeper ACLs | `NoAuthException` | `## NoAuthException for ZooKeeper Connection` |
| KRaft migration controller not ready | `kraft-controller` missing or not Ready | `## Kraft Migration Controller Is Not Ready` |
| KRaft controller cannot write the data volume | `Permission denied` creating `/var/opt/kafka/data/3000` | `## Kraft Migration Controller Pod Fails with Permission Denied on Data Directory` |
| Compacted topic dirty-offset warning repeats | `Resetting first dirty offset` | `## Repeated Occurrence of Warn Log: "Resetting first dirty offset of \[Partition\] is invalid"` |
| Clients stuck rebalancing after broker start | `Preparing to rebalance group` | `## Long Rebalance For Consumer Groups after Starting Kafka Brokers` |
| Brokers slow on NFS | NFS storage, high latency | `## Kafka and NFS. Performance Degradation` |
| Brokers cannot read files on NFS volumes | NFS, files unreadable until the OS reads them | `## Kafka and NFS. Invalid Reading Files on Persistent Volumes` |
| Broker not ready, unsafe memory access | `java.lang.InternalError` in `OffsetIndex.lookup` | `## Kafka Pod Is Not Ready With java.lang.InternalError` |
| Produce fails min.insync.replicas | `NotEnoughReplicasException` | `## Topics with Insufficient Replication Factor` |
| One broker takes all the CPU | All partition leaders on one node | `## Kafka CPU is Overloaded only for one of the Cluster Nodes` |
| AKHQ or CLI cannot describe groups | `This server does not host this topic-partition` | `## Displaying Consumer Groups Fails with "This server does not host this topic-partition"` |
| Topic id disagrees with ZooKeeper | `UNKNOWN_TOPIC_OR_PARTITION`, inconsistent topic ID | `## Synchronization problem between Kafka and ZooKeeper cluster` |
| ZooKeeper and Kafka storage must be restored | Volumes cleared, restore from backup | `## Restore Storage for ZooKeeper and Kafka` |
| Under-replicated or heaviest partitions | ISR smaller than replication factor | `## Dealing with Kafka's Under-replicated Partitions or Heaviest Partitions in Topics Due to Data Sync Issues` |
| Broker cannot register in ZooKeeper | `node already exists` at `/brokers/ids` | `## Kafka Pods Fail with "node already exists" Error` |
| Whole cluster slow, partition count high | `KafkaPartitionCountAlert` | `## Kafka Works Slowly and Consumes a lot of CPU For All Nodes` |
| Topics still have replicas on removed brokers | `Leader: none` after scale-in | `## Kafka Unhealthy After Cluster Scale-In` |
| Monitoring or auxiliary container refused at start | `container has runAsNonRoot and image will run as root` | `## Container Failed with Error: container has runAsNonRoot and image will run as root` |
| Pod evicted for node disk | `ephemeral-storage` | `## Pod Evicted with Error: The node was low on resource: ephemeral-storage` |
| CRD rejected on OpenShift 3.11 | `kmmconfigs.netcracker.com` openAPIV3Schema | `## CRD Creation Failed on OpenShift 3.11` |

When several rows match, read the more specific section first: start failure (`runAsNonRoot`, KRaft `Permission denied`, `ephemeral-storage`, CRD), then disk or memory, then ZooKeeper connection (`ZooKeeperClientTimeoutException`, `ConnectionLoss`, `NoAuthException`), then partition data (`NotEnoughReplicasException`, dirty offset, scale-in, topic id), then degraded or down.

## Assessment

Report the section header, the evidence that selected it, the impact stated in that section, and the resolution steps from that section only. Say which check the ticket, pipeline log, or cluster could not answer.
