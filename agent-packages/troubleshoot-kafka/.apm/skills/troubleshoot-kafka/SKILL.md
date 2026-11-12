---
name: troubleshoot-kafka
description: Diagnose and resolve issues in Apache Kafka clusters.
---

# Troubleshoot Kafka

Use this skill when investigating issues related to:

- **kafka-service-operator** reconciliation of the `Kafka` custom resource (`netcracker.com/v1`, chart `operator/charts/helm/kafka`, deployment `<name>-operator`, `OPERATOR_MODE=kafka`)
- **Kafka brokers** — identical broker Deployments `<name>-<brokerId>` with container `kafka`. There are no master, data, arbiter, or client roles
- **KRaft** (`kafka.kraft`) and the migration controller Deployment `<name>-kraft-controller`
- **ZooKeeper** connectivity, ACLs, and broker registration when the cluster is not in KRaft
- **Storage** on the broker data volume `/var/opt/kafka/data`, and backup files under `/opt/kafka/backup-storage` (a different volume)
- **Resources** — pod CPU and memory, `kafka.heapSize`, consumer lag, partition count
- **KafkaService** (`netcracker.com/v7`, chart `operator/charts/helm/kafka-service`, deployment `<name>-service-operator`, `OPERATOR_MODE=kafkaservice`): monitoring, lag exporter, AKHQ, Mirror Maker, backup daemon, Cruise Control
- **KafkaUser** authentication and the operator-managed ACL role

Failure-mode facts and resolution steps live in `references/troubleshooting.md`. This file routes to those sections and states which Helm values survive a redeploy. Use only the bundled reference. Do not read it in full for a single symptom.

`<name>` is `global.name` (default `kafka`).

1. Grep headers with line numbers first: `grep -n '^## ' references/troubleshooting.md`
2. Match the symptom against the jump table below, or the raw headers if the table is stale.
3. Read from the matched header through the next header of the same or higher level. A `##` section includes its `###` children; stop at the next `##` or `#`. Never load the whole file for one lookup.

Read one more section the same way only when the section already loaded points at it:

- Kafka Memory Limit → `## Memory limit procedure`
- Kafka Disk Filled on All Nodes → `## Disk filled procedure`

The bundled reference appends those two procedures. The public guide links them as scenario files. Do not edit either guide from this skill.

## Component overview

### Charts

Kafka is two Helm charts, installed separately. Install the `kafka` chart before `kafka-service`, except when `global.externalKafka.enabled` is `true`: then the `kafka` chart does not render a `Kafka` CR, and only `kafka-service` is installed.

| Chart | What it installs |
| --- | --- |
| `operator/charts/helm/kafka` | Operator in `kafka` mode and one `Kafka` CR |
| `operator/charts/helm/kafka-service` | Operator in `kafkaservice` mode, one `KafkaService` CR, and the services below |

`kafka-service` services and chart defaults:

| Service | Helm flag | Default |
| --- | --- | --- |
| Monitoring | `monitoring.install` | `true` |
| Lag exporter (inside monitoring) | `monitoring.lagExporter.enabled` | `false` |
| AKHQ | `akhq.install` | `true` |
| Mirror Maker | `mirrorMaker.install` | `false` |
| Mirror Maker monitoring | `mirrorMakerMonitoring.install` | `false` |
| Backup daemon | `backupDaemon.install` | `false` |
| Cruise Control | `cruiseControl.install` | `false` |
| Integration tests | `integrationTests.install` | `false` |

Deployments the operator creates from the `KafkaService` name: `<name>-monitoring`, `<name>-mirror-maker`, `<name>-backup-daemon`, `<name>-cruise-control`, `<name>-lag-exporter` (container `kafka-lag-exporter`). The AKHQ deployment name is `akhq`.

CRDs ship with the charts (`operator/charts/helm/kafka/crds`, `operator/charts/helm/kafka-service/crds`). Creating them requires `CustomResourceDefinition` rights. When the deployer does not have those rights, follow Deployment Permissions in `docs/public/installation.md` before install. That guide also covers Kubernetes below 1.19, which needs a manual CRD install and Helm CRD creation turned off. Do not invent separate `kafka-crd` or `kafka-services-crd` applications; this repository does not define them.

A direct edit on a running pod or Deployment is a workaround. The operator rebuilds broker env from the `Kafka` CR on the next reconcile, and the container rewrites `server.properties` from `CONF_KAFKA_*` on every start.

### Broker defaults

Chart `values.yaml` defaults, before a resource profile replaces them (`operator/charts/helm/kafka/resource-profiles/`):

- `kafka.replicas` 3. Client port 9092. Data volume `kafka.storage.size` 1Gi mounted at `/var/opt/kafka/data`. The template falls back to `2Gi` only when size is empty.
- `kafka.heapSize` 256. The operator sets `HEAP_OPTS` to `-Xms<heapSize>m -Xmx<heapSize>m`. Memory request 600Mi, memory limit 800Mi (the template falls back to a 700Mi limit only when the limit is empty).
- `kafka.zookeeperConnect` `zookeeper:2181`.
- `kafka.kraft.enabled` is `false` in values. The `kraft.enabled` helper still forces KRaft when the image variant is 4, and the default image is Kafka 4 (`kafka.dockerImage`). ZooKeeper mode needs a Kafka 3 image and `kafka.kraft.enabled: false`.
- `kafka.environmentVariables` already contains `CONF_KAFKA_AUTO_CREATE_TOPICS_ENABLE=false`. Helm replaces the whole list.

Backup files live under `/opt/kafka/backup-storage` on the backup daemon, and only when `backupDaemon.install` is `true`. That path is not the broker data volume.

On a cluster that forces non-root: monitoring needs `securityContext.runAsUser: 1000`. KRaft migration controller volume writes need `securityContext.fsGroup: 1000` (`INFRA_KAFKA_FS_GROUP` when cloud integration is enabled). Do not hardcode `fsGroup` where the platform assigns the group, for example OpenShift SCC.

### Critical configuration paths

| Helm value | Chart | Purpose |
| --- | --- | --- |
| `kafka.config` | `kafka` | `server.properties` keys. Map keys are the Apache Kafka property names; values are quoted strings |
| `kafka.environmentVariables` | `kafka` | `CONF_KAFKA_<PROPERTY>=<value>`. Same properties, older form. Overrides `kafka.config` for the same key |
| `kafka.heapSize` | `kafka` | Broker heap in megabytes. Not a `javaOpts` string |
| `kafka.resources` | `kafka` | Broker CPU and memory requests and limits |
| `kafka.storage.size` | `kafka` | Broker PVC size. The operator restarts the broker after a PVC resize |
| `kafka.replicas` | `kafka` | Broker count. Drives the default replication factor |
| `kafka.zookeeperConnect` | `kafka` | ZooKeeper connect string. Not `zookeeper.connect` under `kafka.config` |
| `kafka.kraft.*` | `kafka` | KRaft and migration |
| `mirrorMaker.config` | `kafka-service` | Common Mirror Maker 2 properties (quoted strings) |
| `mirrorMaker.environmentVariables` | `kafka-service` | Common, per-cluster, and per-flow Mirror Maker properties |
| `mirrorMaker.heapSize` | `kafka-service` | Mirror Maker heap in megabytes. Default 256 |
| `akhq.heapSize` | `kafka-service` | AKHQ heap in megabytes. `0` means half of `akhq.resources.requests.memory` |
| `monitoring.thresholds.*` | `kafka-service` | Alert thresholds (lag, partition count, skew). Not broker properties |
| `KafkaUser.spec.authorization.role` | CR | `admin`, `namespace-admin`, `namespace-producer`, or `namespace-consumer` |

#### `server.properties`

Set broker properties under `kafka.config`. Quote every value. The CR field accepts only strings.

```yaml
kafka:
  config:
    num.io.threads: "16"
    log.retention.hours: "168"
    replica.lag.time.max.ms: "30000"
    group.initial.rebalance.delay.ms: "3000"
    min.insync.replicas: "2"
```

The entrypoint writes `CONF_KAFKA_*` into `server.properties` on every start. These keys are overwritten or derived there, so setting them under `kafka.config` does not stick:

- `broker.id`, `log.dirs`, `zookeeper.connect`, and `delete.topic.enable` (`delete.topic.enable` is forced to `true`)
- `offsets.topic.replication.factor` and `transaction.state.log.replication.factor` (both follow the replication factor)
- `default.replication.factor`, when unset, is the broker replica count capped at 3. Setting `default.replication.factor` under `kafka.config` is the way to change it, and the internal-topic factors follow that value
- `min.insync.replicas`, `replica.lag.time.max.ms` (default 30000), and `group.initial.rebalance.delay.ms` (default 3000) keep a value from `kafka.config`

`kafka.environmentVariables` wins over `kafka.config` for the same property. Repeating `environmentVariables` drops list entries you do not copy, including `CONF_KAFKA_AUTO_CREATE_TOPICS_ENABLE=false`.

There is no `KafkaTopic` custom resource. Topic settings are not chart values.

## Forbidden actions

Never recommend these as a configuration fix. The operator or the container start undoes them:

- `kafka-configs.sh --alter` / `--add-config` with `--entity-type brokers`. Broker properties belong in `kafka.config` (or `kafka.environmentVariables` on releases before 2.9.0).
- Editing `server.properties` or `HEAP_OPTS` on a live pod or Deployment. Reconcile and the next start rewrite them from the CR.
- `kafka-acls.sh --add` / `--remove` for a principal owned by a `KafkaUser`. Change `spec.authorization.role` and let the operator reconcile the ACLs.
- Setting `delete.topic.enable`, `broker.id`, `log.dirs`, `zookeeper.connect`, `offsets.topic.replication.factor`, or `transaction.state.log.replication.factor` under `kafka.config`. The entrypoint overwrites them.

Do not invent a `KafkaTopic` CR, a `javaOpts` value, or master/data/arbiter/client Helm paths. This chart does not have them.

These deletes stay limited to the loaded section:

- Do not `rm -rf` a partition directory under `/var/opt/kafka/data` unless the loaded section is Topics with Insufficient Replication Factor, after rebalance failed, or Kafka Unhealthy After Cluster Scale-In, for topics marked for deletion. That delete drops the partition data.
- Do not `rm -f` a segment `.log` unless the loaded section is the dirty-offset warning and the file is a zero-byte log named in that warning.
- Do not run `kafka-partition-logs.sh --delete` unless the loaded section is the under-replicated or heaviest partitions section, or Disk filled procedure. Do not use it on a compacted topic.
- Do not set `unclean.leader.election.enable=true` unless the loaded section is Kafka Unhealthy After Cluster Scale-In. Remove it after leaders are rebalanced.
- Do not `deleteall /admin/delete_topics` unless that same scale-in section already showed the topic there. Delete only the broken topics.
- Do not treat a full `/opt/kafka/backup-storage/` as a full broker data volume, or the reverse.

`kafka-topics.sh --describe` and `kafka-configs.sh` **without** `--alter` are read-only checks already used by the guide. `kafka-configs.sh --alter --entity-type topics` is allowed only for the topic and the keys the loaded section names (`cleanup.policy`, `retention.ms`, `unclean.leader.election.enable`).

## Common failure modes

These are configuration guardrails. Resolution steps stay in the jump table section.

### ISR shrinkage / under-replicated partitions

Symptom: ISR smaller than the replication factor, `NotEnoughReplicasException`, producers stall. Sections: `## Topics with Insufficient Replication Factor`, `## Dealing with Kafka's Under-replicated Partitions or Heaviest Partitions in Topics Due to Data Sync Issues`, `## Kafka is Degraded`.

Check broker readiness and disk before changing replication settings. `replica.lag.time.max.ms` is a `kafka.config` key (entrypoint default 30000). Raising it hides a slow replica; it does not replace a dead broker or a full disk. `default.replication.factor` follows `kafka.replicas` capped at 3 unless `kafka.config` sets it. `min.insync.replicas` is derived from that factor (`1` when the factor is 2) unless `kafka.config` sets it.

### Consumer-group rebalance storms

Symptom: `Preparing to rebalance group`, lag growing. Section: `## Long Rebalance For Consumer Groups after Starting Kafka Brokers`.

The broker knob is `group.initial.rebalance.delay.ms` under `kafka.config` (entrypoint default 3000). `session.timeout.ms` and `heartbeat.interval.ms` are consumer client settings, not `kafka.config`.

### Retention vs disk usage

Symptom: `No space left on device` writing `/var/opt/kafka/data`, brokers refuse writes. Sections: `## Data Is Out of Space`, then `## Disk filled procedure`.

Broker retention is `log.retention.hours`, `log.retention.bytes`, `log.segment.bytes`, and `log.roll.hours` under `kafka.config`. The guide's 85% target is the steady-state goal. Kafka deletes passive segments only; lowering retention does not delete the active segment until it rolls. If the disk is already full, the disk procedure says the broker stays down until passive segments are removed.

PVC capacity is `kafka.storage.size`, not a `server.properties` key. The operator restarts the broker after the volume controller finishes the resize. Do not assume every storage class allows expansion.

Pod memory and CPU are `kafka.resources` plus `kafka.heapSize`. Alert limits such as `monitoring.thresholds.lagAlert` and `monitoring.thresholds.partitionCountAlert` live on the `kafka-service` chart. They do not change broker retention or heap.

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

## Log analysis

Run read-only commands. `<ns>` is the release namespace. `<name>` defaults to `kafka`.

```bash
kubectl logs -n <ns> deploy/<name>-operator | grep -iE "error|failed|reconcile"
kubectl logs -n <ns> deploy/<name>-service-operator | grep -iE "error|failed|reconcile"
kubectl logs -n <ns> deploy/<name>-1 -c kafka | grep -E "No space left on device|Java heap space|ZooKeeperClientTimeoutException|ConnectionLoss|NoAuthException|node already exists|NotEnoughReplicasException|Preparing to rebalance group|Permission denied"
kubectl get kafka,kafkaservice,kafkauser,pods,pvc -n <ns>
```

Mirror Maker: `deploy/<name>-mirror-maker`. Monitoring: `deploy/<name>-monitoring`. Backup daemon: `deploy/<name>-backup-daemon`. Inside a broker, `kafka-topics.sh --describe` is the read-only check the guide uses. Do not add `--alter`.

## Assessment

Report the section header, the evidence that selected it, the impact stated in that section, and the resolution steps from that section only. Say which check the ticket, pipeline log, or cluster could not answer.

### Recommendation discipline

- Phrase each `server.properties` change as: set `<key>: "<value>"` under `kafka.config` in the `kafka` chart and redeploy through the normal delivery channel. Put the YAML snippet under that sentence.
- Phrase broker heap as: set `kafka.heapSize: <megabytes>` in the `kafka` chart and redeploy. Raise `kafka.resources.requests.memory` and `kafka.resources.limits.memory` so the heap fits. Mirror Maker heap is `mirrorMaker.heapSize` on the `kafka-service` chart. AKHQ heap is `akhq.heapSize` (`0` means half the memory request).
- Phrase Mirror Maker properties as: set `<key>: "<value>"` under `mirrorMaker.config` for a common property, or `mirrorMaker.environmentVariables` for a per-cluster or per-flow property, and redeploy the `kafka-service` chart.
- Phrase a `KafkaUser` ACL change as a change to `spec.authorization.role`, not a `kafka-acls.sh` command.
- If the loaded section still shows `CONF_KAFKA_*` on the Deployment, translate that into `kafka.config` for releases since 2.9.0. Do not tell the operator to edit the live Deployment.
- If no chart value or CR field documents the change, omit the recommendation. Do not fall back to `kafka-configs.sh --alter` on brokers.

## When to escalate

Escalate to component maintainers when the loaded section was followed and one of these remains:

- The cluster is still degraded or down, or the operator pod crash-loops on reconcile.
- ZooKeeper and Kafka volumes were both cleared and the restore section did not recover the cluster.
- The KRaft migration controller stays not Ready after the readiness and permission-denied sections.
- The data volume is still full after the disk procedure, including passive-segment cleanup.
- Scale-in left partitions with `Leader: none` after the scale-in section.
- A `KafkaUser` stays in authorization `failure` after the role and secret match the CRD.

Include the matched section header, operator logs, logs of the affected broker container `kafka`, and `kubectl get kafka,kafkaservice,kafkauser,pods,pvc -n <ns>`. For Mirror Maker or backup, include that deployment's logs.
