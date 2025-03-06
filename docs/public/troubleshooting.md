[[_TOC_]]

## ZooKeeper Failure

### Description

This problem can be detected on the ZooKeeper monitoring dashboard. It is indicated by a "failed" cluster status.

Kafka uses ZooKeeper for controller election, cluster membership management, topic configuration, and storing ACL information.

For more information about Kafka data structures, refer to
[https://cwiki.apache.org/confluence/display/KAFKA/Kafka+data+structures+in+Zookeeper].

These functions are affected in case of ZooKeeper failure.

For Kafka users, it means that it is impossible to create new topics or alter existing ones.
However, clients that use new consumer API are still be able to send messages to existing topics and read from them.

Kafka automatically attempts to reconnect to ZooKeeper when the connection is lost. This means that
this problem is automatically resolved by Kafka as soon as ZooKeeper service becomes available again.

### Alerts

Not applicable

### Stack trace

```text
[2024-08-06T04:53:40,367][WARN][category=org.apache.zookeeper.ClientCnxn] Session 0x3009a99fa94357f for server zookeeper.zookeeper-service/172.30.239.30:2181, Closing socket connection. Attempting reconnect except it is a SessionExpiredException.
EndOfStreamException: Unable to read additional data from server sessionid 0x3009a99fa94357f, likely server has closed socket
 at org.apache.zookeeper.ClientCnxnSocketNIO.doIO(ClientCnxnSocketNIO.java:77)
 at org.apache.zookeeper.ClientCnxnSocketNIO.doTransport(ClientCnxnSocketNIO.java:350)
 at org.apache.zookeeper.ClientCnxn$SendThread.run(ClientCnxn.java:1289)
```

### How to solve

To identify the reason for ZooKeeper failure and suggested actions for restoration, refer to [Zookeeper Troubleshooting Guide](https://github.com/Netcracker/qubership-zookeeper/tree/main/docs/public/troubleshooting.md).

### Recommendations

Not applicable

## Partition Leader Crash

### Description

This problem can be detected on the Kafka monitoring dashboard. If the current cluster's size
goes down, then it means that one of the nodes most likely has crashed.

Crashing of the partition leader in Kafka cluster may have different effects on availability
of the Kafka cluster based on its current condition and partition/topic configuration.

In general, when a broker fails, partitions with a leader on that broker become
temporarily unavailable. Kafka automatically moves the leader of those
unavailable partitions to some other replicas to continue serving the client requests.
The observed unavailability time is proportional to the number of partitions. To minimize
the downtime of the service, it is recommended to minimize the amount of partitions per broker.

Also, it is possible that the failed broker was the controller. In this case, the process
of electing the new leaders does not start until the controller fails over to a new broker.
The controller failover happens automatically, but requires the new controller to read
some metadata for every partition from ZooKeeper during initialization. It will also
extend the unavailability window.

Any replica in the ISR set is eligible to be elected as leader. If there are no replicas in the
ISR set, then the partition will be unavailable until the crashed leader of partition goes back up.
**Note**: This can be changed by setting the property `unclean.leader.election.enable` to `true`
in the configuration. For more information about Kafka configuration,
refer to Kafka Documentation located at [https://kafka.apache.org/0110/documentation.html#topicconfigs](https://kafka.apache.org/0110/documentation.html#topicconfigs).
In this case, replicas that are not in the ISR set can be elected as leader
as a last resort, even though doing so may result in data loss.

### Alerts

Not applicable

### Stack trace

Not applicable

### How to solve

If the crashed broker is unable to start or rejoin the cluster, then check monitoring for other problems
that could have occurred at the same time. When the problem is localized, go to the
appropriate problem description and troubleshooting procedure to fix it.

### Recommendations

Not applicable

## Data Is Out of Space

### Description

If the disk approaches 100% utilization, you must adjust the number of days that data is being
retained. An optimal value of data to retain should be 85% or less of the disk capacity.

For more information, see [Kafka Disk Filled on All Nodes](scenarios/kafka_disk_filled_on_all_nodes.md).

### Alerts

Not applicable

### Stack trace

```text
[2022-02-03T07:06:32,472][ERROR][category=kafka.server.LogDirFailureChannel] Error while writing to checkpoint file /var/opt/kafka/data/2/__consumer_offsets-42/leader-epoch-checkpoint
java.io.IOException: No space left on device
at java.base/java.io.FileOutputStream.writeBytes(Native Method)
at java.base/java.io.FileOutputStream.write(Unknown Source)
at java.base/sun.nio.cs.StreamEncoder.writeBytes(Unknown Source)
```

### How to solve

Adjust the number of days that data is being retained.

### Recommendations

An optimal value of data to retain should be 85% or less of the disk capacity.

## Kafka is Degraded

### Description

A `Degraded` status on monitoring indicates that at least one of the Kafka brokers have failed.

Each partition has one server which acts as the "leader" and zero or more servers which act as "followers".
Whenever a broker stops or crashes, leadership for that broker's partitions transfers to other replicas.
By default, the Kafka cluster will try to restore leadership to the restored replicas.
For more information about leader balancing in Kafka,
refer to the official Kafka documentation at [https://kafka.apache.org/documentation/#basic_ops_leader_balancing](https://kafka.apache.org/documentation/#basic_ops_leader_balancing).

To identify the reason for the node failure, check the monitoring dashboard for any other problems
that may have occurred around the same time the node failed. When the problem is localized,
go to the appropriate problem description and troubleshooting procedure to fix it.
Once the node is fixed, it will rejoin ensemble as a follower node.

### Alerts

* [KafkaIsDegradedAlert](./alerts.md#kafkaisdegradedalert)

### Stack trace

```text
2024-08-06T06:54:18,617][INFO][category=org.apache.kafka.clients.NetworkClient] [Producer clientId=CruiseControlMetricsReporter] Node 3 disconnected.
[2024-08-06T06:54:18,618][WARN][category=org.apache.kafka.clients.NetworkClient] [Producer clientId=CruiseControlMetricsReporter] Connection to node 3 (kafka-3.kafka-service/172.30.110.170:9092) could not be established. Node may not be available.
```

### How to solve

1. Restart or redeploy Kafka pods if they are in a failed state.
2. Investigate and address any resource constraints affecting the Kafka pod performance.

### Recommendations

Not applicable

## Kafka is Down

### Description

A `Down` status on monitoring indicates that all the Kafka brokers have failed.

### Alerts

* [KafkaIsDownAlert](./alerts.md#kafkaisdownalert)

### Stack trace

Not applicable

### How to solve

Try to reboot all Kafka Service deployments.

### Recommendations

Not applicable

## Kafka Mirror Maker is Degraded

### Description

A `Degraded` status on monitoring indicates that one of the Kafka Mirror Maker node has failed.

A possible reason - left or right part of Disaster Recovery schema has failed. For example, right Kafka is
in `Down` status. It leads failed status for "right" Kafka mirror maker node.

### Alerts

* [KafkaMirrorMakerIsDegradedAlarm](./alerts.md#kafkamirrormakerisdegradedalarm)

### Stack trace

Not applicable

### How to solve

Try to up Kafka Service and reboot appropriate Kafka Mirror Maker.

### Recommendations

Not applicable

## Kafka Mirror Maker is Down

### Description

A `Down` status on monitoring indicates that all the Kafka Mirror Maker nodes have failed.

A possible reason - left and right part of Disaster Recovery schema have failed ("left" and "right" Kafka in the
`Down` status).

### Alerts

* [KafkaMirrorMakerIsDownAlarm](./alerts.md#kafkamirrormakerisdownalarm)

### Stack trace

Not applicable

### How to solve

Try to up all Kafka Services and reboot Kafka Mirror Maker pods.

### Recommendations

Not applicable

## CPU Limit Reached

### Description

Kafka request processing may be impacted, including up to potential node failure, when CPU consumption
reaches the resource limit. You may need to increase CPU resource requests and limits.

### Alerts

* [KafkaCPUUsageAlert](./alerts.md#kafkacpuusagealert)

### Stack trace

Not applicable

### How to solve

You may need to increase CPU resource requests and limits.

### Recommendations

Not applicable

## Memory Limit Reached

### Description

Kafka request processing may be impacted, including up to potential node failure, when memory consumption
reaches the resource limit. You may need to increase resource requests and limits or scale out the cluster.

For more information, see [Kafka Memory Limit](scenarios/kafka_memory_limit.md).

### Alerts

* [KafkaMemoryUsageAlert](./alerts.md#kafkamemoryusagealert)
* [KafkaHeapMemoryUsageAlert](./alerts.md#kafkaheapmemoryusagealert)
* [KafkaGCCountAlert](./alerts.md#kafkagccountalert)

### Stack trace

```text
[2018-04-06T11:02:29,332][ERROR][category=kafka.coordinator.group.GroupMetadataManager] [Group Metadata Manager on Broker 1]: Error loading offsets from __consumer_offsets-27
java.lang.OutOfMemoryError: Java heap space
	at java.nio.HeapByteBuffer.<init>(HeapByteBuffer.java:57)
	at java.nio.ByteBuffer.allocate(ByteBuffer.java:335)
	at kafka.coordinator.group.GroupMetadataManager.buffer$lzycompute$1(GroupMetadataManager.scala:477)
	at kafka.coordinator.group.GroupMetadataManager.buffer$1(GroupMetadataManager.scala:477)
	at kafka.coordinator.group.GroupMetadataManager.loadGroupsAndOffsets(GroupMetadataManager.scala:491)
	at kafka.coordinator.group.GroupMetadataManager.kafka$coordinator$group$GroupMetadataManager$$doLoadGroupsAndOffsets$1(GroupMetadataManager.scala:455)
	at kafka.coordinator.group.GroupMetadataManager$$anonfun$loadGroupsForPartition$1.apply$mcV$sp(GroupMetadataManager.scala:441)
	at kafka.utils.KafkaScheduler$$anonfun$1.apply$mcV$sp(KafkaScheduler.scala:110)
	at kafka.utils.CoreUtils$$anon$1.run(CoreUtils.scala:57)
	at java.util.concurrent.Executors$RunnableAdapter.call(Executors.java:511)
	at java.util.concurrent.FutureTask.run(FutureTask.java:266)
	at java.util.concurrent.ScheduledThreadPoolExecutor$ScheduledFutureTask.access$201(ScheduledThreadPoolExecutor.java:180)
	at java.util.concurrent.ScheduledThreadPoolExecutor$ScheduledFutureTask.run(ScheduledThreadPoolExecutor.java:293)
	at java.util.concurrent.ThreadPoolExecutor.runWorker(ThreadPoolExecutor.java:1142)
	at java.util.concurrent.ThreadPoolExecutor$Worker.run(ThreadPoolExecutor.java:617)
	at java.lang.Thread.run(Thread.java:745)
```

### How to solve

You may need to increase resource requests and limits or scale out the cluster.

### Recommendations

Not applicable

## Lag Limit Reached

### Description

The Kafka data can be lost because its persistence is based on retention.

### Alerts

* [KafkaLagAlert](./alerts.md#kafkalagalert)

### Stack trace

Not applicable

### How to solve

There are several options for resolving the issue:

* Increase the number of topic partitions and the number of consumers.
* Increase resource requests and limits. Recommend to check the [performance guide](./performance.md).

### Recommendations

Not applicable

## Timed Out Waiting for ZooKeeper Connection

### Description

Kafka fails with the following error:

```text
kafka.zookeeper.ZooKeeperClientTimeoutException: Timed out waiting for connection while in state: CONNECTING
```

Since Kafka cannot connect to ZooKeeper, you need to make sure your ZooKeeper cluster is fully operated.
To identify the reason for ZooKeeper failure and suggested actions for restoration, refer to [Zookeeper Troubleshooting Guide](https://github.com/Netcracker/qubership-zookeeper/tree/main/docs/public/troubleshooting.md).

### Alerts

Not applicable

### Stack trace

```text
[2018-08-21T07:34:17,407][ERROR][category=kafka.server.KafkaServer] Fatal error during KafkaServer startup. Prepare to shutdown
kafka.zookeeper.ZooKeeperClientTimeoutException: Timed out waiting for connection while in state: CONNECTING
[2018-08-21T08:04:12,258][INFO][category=kafka.zookeeper.ZooKeeperClient] [ZooKeeperClient] Waiting until connected.
```

### How to solve

If ZooKeeper works fine you need to increase the connection timeout to ZooKeeper:

1. Edit the environment variables of the Kafka deployment configuration.
2. Set the `CONF_KAFKA_ZOOKEEPER_CONNECTION_TIMEOUT_MS` environment variable to `30000`.
3. Save your changes.

By default, Kafka has the following settings:

```text
# Timeout in ms for connecting to zookeeper                                                                                                                                                            
zookeeper.connection.timeout.ms=6000
```

### Recommendations

Not applicable

## ConnectionLoss Error for ZooKeeper Connection

### Description

When Kafka pods fail or do not respond on requests with the following logs:

```text
Client session timed out, have not heard from server in 30004ms for sessionid 0x0, closing socket connection and attempting reconnect
```

The error log depicts that Kafka cannot connect to ZooKeeper because it does not work as expected.
It could happen after restarting ZooKeeper leader or OpenShift/Kubernetes nodes.

### Alerts

Not applicable

### Stack trace

```text
[2020-08-06T12:47:06,278][INFO][zk_id=zookeeper:2181][thread=main-SendThread(zookeeper:2181)][class=ClientCnxn$SendThread@1238] Client session timed out, have not heard from server in 30004ms for sessionid 0x0, closing socket connection and attempting reconnect                                                                                                                                     
KeeperErrorCode = ConnectionLoss for 
```

### How to solve

To identify the reason for ZooKeeper failure and suggested actions for restoration, refer to [Zookeeper Troubleshooting Guide](https://github.com/Netcracker/qubership-zookeeper/tree/main/docs/public/troubleshooting.md).

### Recommendations

Not applicable

## NoAuthException for ZooKeeper Connection

### Description

When Kafka pods fail with following logs:

```text
org.apache.zookeeper.KeeperException$NoAuthException: KeeperErrorCode = NoAuth for /config/users
```

Possible issue is when Kafka's Znode in ZooKeeper has ACL which Kafka currently cannot access to.

### Alerts

Not applicable

### Stack trace

```text
Error while executing topic command org.apache.zookeeper.KeeperException$NoAuthException: KeeperErrorCode = NoAuth for /config/topics
org.I0Itec.zkclient.exception.ZkException: org.apache.zookeeper.KeeperException$NoAuthException: KeeperErrorCode = NoAuth for /config/topics
 at org.I0Itec.zkclient.exception.ZkException.create(ZkException.java:68)
 at org.I0Itec.zkclient.ZkClient.retryUntilConnected(ZkClient.java:685)
 at org.I0Itec.zkclient.ZkClient.create(ZkClient.java:304)
 at org.I0Itec.zkclient.ZkClient.createPersistent(ZkClient.java:213)
 at kafka.utils.ZkUtils$.createParentPath(ZkUtils.scala:215)
 at kafka.utils.ZkUtils$.updatePersistentPath(ZkUtils.scala:338)
 at kafka.admin.AdminUtils$.writeTopicConfig(AdminUtils.scala:247)
```

### How to solve

Execute following commands from any ZooKeeper pod.

1. Create the JAAS configuration file:

    ```sh
    cat >> ${ZOOKEEPER_HOME}/conf/client_jaas.conf << EOL
    Client {
               org.apache.zookeeper.server.auth.DigestLoginModule required
               username="username"
               password="password";
        };
    EOL
    ```

   Note that ZooKeeper admin's username and password should be used.

2. Specify path to the JAAS configuration file in the JVM property `java.security.auth.login.config`.

   For example, for `zkCli.sh` client, it is necessary to use the following command:

    ```sh
    export CLIENT_JVMFLAGS=-"Djava.security.auth.login.config=${ZOOKEEPER_HOME}/conf/client_jaas.conf" && ./bin/zkCli.sh
    ```

3. Set the permissions for `/config` and `/broker` znodes for any user.

   ```text
   setAcl -R /config world:anyone:cdrwa
   setAcl -R /brokers world:anyone:cdrwa 
   ```

4. Restart Kafka pods.

### Recommendations

Not applicable

## Repeated Occurrence of Warn Log: "Resetting first dirty offset of \[Partition\] is invalid"

### Description

When Kafka brokers have many warnings like the following:

```text
[2019-12-13T13:42:28,581][WARN][category=kafka.log.LogCleanerManager$] Resetting first dirty offset of event-aggregator-develop.ea.aggregate-freezer-develop.ea.streaming.frozenaggregates.table-changelog-5 to log start offset 4504225 since the checkpointed offset 3959401 is invalid.
[2019-12-13T13:42:28,581][WARN][category=kafka.log.LogCleanerManager$] Resetting first dirty offset of event-aggregator-develop.ea.aggregate-freezer-develop.ea.streaming.frozenaggregates.table-changelog-8 to log start offset 5558608 since the checkpointed offset 5177936 is invalid.
[2019-12-13T13:42:28,582][WARN][category=kafka.log.LogCleanerManager$] Resetting first dirty offset of event-aggregator-develop.ea.aggregate-freezer-develop.ea.streaming.frozenaggregates.table-changelog-14 to log start offset 3764541 since the checkpointed offset 3763746 is invalid.
```

It might mean there are empty-size log files for partitions of the topic.
Log Cleaner Manager will try to reset the offset every time for this topic, and it works incorrectly.

This issue is described in [KAFKA-6266](https://issues.apache.org/jira/browse/KAFKA-6266), but there is no permanent solution.

### Alerts

Not applicable

### Stack trace

```text
[2019-12-13T13:42:28,581][WARN][category=kafka.log.LogCleanerManager$] Resetting first dirty offset of event-aggregator-develop.ea.aggregate-freezer-develop.ea.streaming.frozenaggregates.table-changelog-5 to log start offset 4504225 since the checkpointed offset 3959401 is invalid.
[2019-12-13T13:42:28,581][WARN][category=kafka.log.LogCleanerManager$] Resetting first dirty offset of event-aggregator-develop.ea.aggregate-freezer-develop.ea.streaming.frozenaggregates.table-changelog-8 to log start offset 5558608 since the checkpointed offset 5177936 is invalid.
[2019-12-13T13:42:28,582][WARN][category=kafka.log.LogCleanerManager$] Resetting first dirty offset of event-aggregator-develop.ea.aggregate-freezer-develop.ea.streaming.frozenaggregates.table-changelog-14 to log start offset 3764541 since the checkpointed offset 3763746 is invalid.
```

### How to solve

To solve this problem, you can apply WA.

First of all, you need to make sure you have exactly this problem:

1. You need to determine invalid log partitions. You can find their names in the log message.
   For example, the message:

    ```text
    Resetting first dirty offset of event-aggregator-develop.ea.aggregate-freezer-develop.ea.streaming.frozenaggregates.table-changelog-5 to log start offset 4504225 since the checkpointed offset 3959401 is invalid. 
    ```

   Contains the name of the invalid partition
   `event-aggregator-develop.ea.aggregate-freezer-develop.ea.streaming.frozenaggregates.table-changelog-5`.

   Also, you can find empty (zero-byte) log files via the command `find /var/opt/kafka/data -size 0`.

2. You need to check whether the file `/var/opt/kafka/data/<broker-id>/cleaner-offset-checkpoint` contains an offset for
   the found invalid partitions. For example:

   ```sh
    __consumer_offsets 33 8361
    __consumer_offsets 23 2096
    event-aggregator-develop.ea.aggregate-freezer-develop.ea.streaming.frozenaggregates.table-changelog 5 3959401
    __consumer_offsets 49 477814
    ```

If you have invalid partitions with empty log files and records in `cleaner-offset-checkpoint` for these partitions,
you need to remove the incorrect log files and restart Kafka broker.

The better way to delete invalid logs is:

1. Stop Kafka broker.
2. Remove all incorrect partition logs that are found in the previous step using `ssh` to connect to Persistent Volume storage.
   Partitions are placed in the `/var/opt/kafka/data/<broker-id>` directory.

   For example:

    ```sh
    rm -f /var/opt/kafka/data/1/event-aggregator-develop.ea.aggregate-freezer-develop.ea.streaming.frozenaggregates.table-changelog-5/00000000000004504225.log
    ```

3. Start Kafka broker.
4. Repeat Steps 1-3 for each Kafka broker where you have faced this issue.

**Note**: If you do not have access to Persistence Volumes directly, you can skip Step 1 and remove files via the terminal.
Then, restart the Kafka pod.

Note that applying this work around leads to downtime due to restarting Kafka brokers and resetting offsets for invalid partitions.

### Recommendations

Not applicable

## Long Rebalance For Consumer Groups after Starting Kafka Brokers

### Description

If your clients cannot connect to Kafka for a long time after restart, and Kafka brokers have many warnings like the following:

```text
kafka_1 | [2019-01-25 16:08:42,279] INFO [GroupCoordinator 1001]: Preparing to rebalance group group1 in state PreparingRebalance with old generation 442 (__consumer_offsets-40) (reason: Adding new member … (kafka.coordinator.group.GroupCoordinator)
kafka_1 | [2019-01-25 16:08:42,281] INFO [GroupCoordinator 1001]: Stabilized group group1 generation 443 (__consumer_offsets-40) (kafka.coordinator.group.GroupCoordinator)
kafka_1 | [2019-01-25 16:08:42,284] INFO [GroupCoordinator 1001]: Assignment received from leader for group group1 for generation 443 (kafka.coordinator.group.GroupCoordinator)
```

It means a rebalance is being performed for the consumer.

A rebalance is typically initiated by the group coordinator when a consumer requests to join a group or leaves a group.
There might be situation when the consumer tries to join a group,
then leaves the group by timeout because it did not receive a response, and then tries to join again.
It can lead to a long or even endless rebalance.

### Alerts

Not applicable

### Stack trace

```text
kafka_1 | [2019-01-25 16:08:42,279] INFO [GroupCoordinator 1001]: Preparing to rebalance group group1 in state PreparingRebalance with old generation 442 (__consumer_offsets-40) (reason: Adding new member … (kafka.coordinator.group.GroupCoordinator)
kafka_1 | [2019-01-25 16:08:42,281] INFO [GroupCoordinator 1001]: Stabilized group group1 generation 443 (__consumer_offsets-40) (kafka.coordinator.group.GroupCoordinator)
kafka_1 | [2019-01-25 16:08:42,284] INFO [GroupCoordinator 1001]: Assignment received from leader for group group1 for generation 443 (kafka.coordinator.group.GroupCoordinator)
```

### How to solve

To avoid this issue, you can adjust the corresponding consumer configurations (on the consumer side):

You can control the session timeout by overriding the `session.timeout.ms` value.
The default is `30` seconds, but you can safely increase it to avoid excessive rebalances
if you find that your application needs more time to process messages.
This concern is mainly relevant if you are using the Java consumer and handling messages in the same thread.
In that case, you may also want to adjust `max.poll.records` to tune the number of records that must be handled on every loop iteration.

The main drawback to using a larger session timeout is that it will take longer for the coordinator
to detect when a consumer instance has crashed,
which means it will also take longer for another consumer in the group to take over its partitions.
For normal shutdowns, however, the consumer sends an explicit request to the coordinator to leave the group,
which triggers an immediate rebalance.

The other setting that affects rebalance behavior is `heartbeat.interval.ms`.
This controls how often the consumer sends heartbeats to the coordinator.
It is also the way that the consumer detects when a rebalance is needed,
so a lower heartbeat interval will generally mean faster rebalances.
The default setting is 3 seconds. For larger groups, it may be wise to increase this setting.

It is recommended to set `session.timeout.ms` to be large enough that commit failures from rebalances are rare.
As previously mentioned, the only drawback to this is a longer delay before partitions can be re-assigned in the event of a hard failure
(where the consumer cannot be cleanly shutdown with `close()`).
This should be rare in practice.

For more information, refer to [Kafka Consumers](https://docs.confluent.io/3.0.0/clients/consumer.html).

You can also adjust the following Kafka Broker's properties:

`group.initial.rebalance.delay.ms`: The amount of time the group coordinator waits for more consumers
to join a new group before performing the first rebalance.
A longer delay means potentially fewer rebalances, but increases the time until processing begins. Default: `3000`.
To change this value, you need to add the environment variable `CONF_KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS` to
the Kafka broker Deployment Configuration.

`group.min.session.timeout.ms`: The minimum allowed session timeout for registered consumers.
Shorter timeouts result in quicker failure detection at the cost of more frequent consumer heartbeating,
which can overwhelm broker resources. Default: `6000`.
To change this value, you need to add the environment variable `CONF_KAFKA_GROUP_MIN_SESSION_TIMEOUT_MS` to
the Kafka broker Deployment Configurations.

For more information, refer to [Kafka Documentations](https://kafka.apache.org/documentation/).

### Recommendations

Not applicable

## Kafka and NFS. Performance Degradation

### Description

Not all types of NFS storage are compatible with Kafka.
Using NFS can lead to several problems, and also it may cause performance degradation.

Usage of shared storage between ZooKeeper and Kafka can also lead to performance degradation.

Thus, it is not recommended to use NFS as main storage for Kafka brokers.

For more information about features of working Kafka with NFS, refer to [Kafka on NFS](kafka-on-nfs.md).

**Note**: There is a general recommendation against running Apache Kafka on NFS storage.

Kafka stores partitions as directories with data files inside it, and it does not free up the file handle before unlinking it.
NFS mounts behave differently. Due to the design of the NFS protocol, there is no way for a file to be deleted from the namespace,
but it still remains in use by an application.
Thus, NFS clients have to emulate this using what already exists in the protocol.

On the next start, the broker knows it did not stop cleanly, so the first step is to re-index the data files it has on disk and remove
the files and partitions it no longer needs.
The broker which is restarted starts loading logs using all the available recovery threads. Other brokers get degradation of disk IO after
that.

For more information, refer to [A Practical Introduction to Kafka Storage Internals](https://medium.com/@durgaswaroop/a-practical-introduction-to-kafka-storage-internals-d5b544f6925f).

### Alerts

Not applicable

### Stack trace

```text
[2020-02-10 12:06:07,494] WARN [ReplicaFetcher replicaId=1, leaderId=3, fetcherId=1] Error in response for fetch request (type=FetchRequest, replicaId=1, maxWait=500, minBytes=1, maxBytes=10485760, fetchData={}, isolationLevel=READ_UNCOMMITTED, toForget=, metadata=(sessionId=1097663787, epoch=13128), rackId=) (kafka.server.ReplicaFetcherThread)
java.io.IOException: Connection to 3 was disconnected before the response was read
	at org.apache.kafka.clients.NetworkClientUtils.sendAndReceive(NetworkClientUtils.java:100)
	at kafka.server.ReplicaFetcherBlockingSend.sendRequest(ReplicaFetcherBlockingSend.scala:107)
	at kafka.server.ReplicaFetcherThread.fetchFromLeader(ReplicaFetcherThread.scala:196)
	at kafka.server.AbstractFetcherThread.kafka$server$AbstractFetcherThread$$processFetchRequest(AbstractFetcherThread.scala:286)
	at kafka.server.AbstractFetcherThread$$anonfun$maybeFetch$1.apply(AbstractFetcherThread.scala:133)
	at kafka.server.AbstractFetcherThread$$anonfun$maybeFetch$1.apply(AbstractFetcherThread.scala:132)
	at scala.Option.foreach(Option.scala:257)
	at kafka.server.AbstractFetcherThread.maybeFetch(AbstractFetcherThread.scala:132)
	at kafka.server.AbstractFetcherThread.doWork(AbstractFetcherThread.scala:114)
	at kafka.utils.ShutdownableThread.run(ShutdownableThread.scala:96)


[2020-02-10 12:06:21,782] ERROR [ReplicaManager broker=1] Error processing append operation on partition dev.ra.input.facts-4 (kafka.server.ReplicaManager)  
```

### How to solve

If you get a lot of log messages like mentioned above but the remote broker is available it can be a reason to use local storage instead of NFS.

### Recommendations

Not applicable

## Kafka and NFS. Invalid Reading Files on Persistent Volumes

### Description

Not all types of NFS storage are compatible with Kafka.
Using NFS can lead to several problems, and also it may cause performance degradation.

Usage of shared storage between ZooKeeper and Kafka can also lead to performance degradation.

Thus, it is not recommended to use NFS as main storage for Kafka brokers.

For more information about features of working Kafka with NFS, refer to [Kafka on NFS](kafka-on-nfs.md).

**Note**: There is a general recommendation against running Apache Kafka on NFS storage.

Kafka brokers may fail with the logs below (in "Stack trace" section) or any other "Invalid argument" error with access to file.

It happens when the Kafka broker tries to get access to the files which were created by previous pod and fails
because these files are not available in the file system.
It depends on NFS type but some services, for example Kafka, cannot access files until they are read by the operation system.

### Alerts

Not applicable

### Stack trace

```text
[2020-03-11T13:11:22,596][ERROR][category=kafka.server.LogDirFailureChannel] Disk error while locking directory /var/opt/kafka/data/1
java.nio.file.FileSystemException: /var/opt/kafka/data/1/.lock: Invalid argument
	at sun.nio.fs.UnixException.translateToIOException(UnixException.java:91)
	at sun.nio.fs.UnixException.rethrowAsIOException(UnixException.java:102)
	at sun.nio.fs.UnixException.rethrowAsIOException(UnixException.java:107)
	at sun.nio.fs.UnixFileSystemProvider.newFileChannel(UnixFileSystemProvider.java:177)
	at java.nio.channels.FileChannel.open(FileChannel.java:287)
	at java.nio.channels.FileChannel.open(FileChannel.java:335)
	at kafka.utils.FileLock.<init>(FileLock.scala:30)
	at kafka.log.LogManager$$anonfun$lockLogDirs$1.apply(LogManager.scala:246)
	at kafka.log.LogManager$$anonfun$lockLogDirs$1.apply(LogManager.scala:244)
	at scala.collection.TraversableLike$$anonfun$flatMap$1.apply(TraversableLike.scala:241)
	at scala.collection.TraversableLike$$anonfun$flatMap$1.apply(TraversableLike.scala:241)
	at scala.collection.mutable.ResizableArray$class.foreach(ResizableArray.scala:59)
	at scala.collection.mutable.ArrayBuffer.foreach(ArrayBuffer.scala:48)
	at scala.collection.TraversableLike$class.flatMap(TraversableLike.scala:241)
	at scala.collection.AbstractTraversable.flatMap(Traversable.scala:104)
	at kafka.log.LogManager.lockLogDirs(LogManager.scala:244)
	at kafka.log.LogManager.<init>(LogManager.scala:105)
	at kafka.log.LogManager$.apply(LogManager.scala:1078)
	at kafka.server.KafkaServer.startup(KafkaServer.scala:250)
	at kafka.server.KafkaServerStartable.startup(KafkaServerStartable.scala:44)
	at kafka.Kafka$.main(Kafka.scala:84)
	at kafka.Kafka.main(Kafka.scala)

[2020-03-11T13:11:22,612][ERROR][category=kafka.server.KafkaServer] [KafkaServer id=1] Fatal error during KafkaServer startup. Prepare to shutdown
java.nio.file.FileSystemException: /var/opt/kafka/data/1/recovery-point-offset-checkpoint: Invalid argument
	at sun.nio.fs.UnixException.translateToIOException(UnixException.java:91)
	at sun.nio.fs.UnixException.rethrowAsIOException(UnixException.java:102)
	at sun.nio.fs.UnixException.rethrowAsIOException(UnixException.java:107)
	at sun.nio.fs.UnixFileSystemProvider.newByteChannel(UnixFileSystemProvider.java:214)
	at java.nio.file.Files.newByteChannel(Files.java:361)
	at java.nio.file.Files.createFile(Files.java:632)
	at kafka.server.checkpoints.CheckpointFile.<init>(CheckpointFile.scala:45)
	at kafka.server.checkpoints.OffsetCheckpointFile.<init>(OffsetCheckpointFile.scala:56)
	at kafka.log.LogManager$$anonfun$8.apply(LogManager.scala:107)
	at kafka.log.LogManager$$anonfun$8.apply(LogManager.scala:106)
	at scala.collection.TraversableLike$$anonfun$map$1.apply(TraversableLike.scala:234)
	at scala.collection.TraversableLike$$anonfun$map$1.apply(TraversableLike.scala:234)
	at scala.collection.mutable.ResizableArray$class.foreach(ResizableArray.scala:59)
	at scala.collection.mutable.ArrayBuffer.foreach(ArrayBuffer.scala:48)
	at scala.collection.TraversableLike$class.map(TraversableLike.scala:234)
	at scala.collection.AbstractTraversable.map(Traversable.scala:104)
	at kafka.log.LogManager.<init>(LogManager.scala:106)
	at kafka.log.LogManager$.apply(LogManager.scala:1078)
	at kafka.server.KafkaServer.startup(KafkaServer.scala:250)
	at kafka.server.KafkaServerStartable.startup(KafkaServerStartable.scala:44)
	at kafka.Kafka$.main(Kafka.scala:84)
	at kafka.Kafka.main(Kafka.scala)
```

### How to solve

To resolve this issue a workaround is recommended.
You need to add environment variable `SCAN_FILE_SYSTEM` with value `true` to each deployment config of Kafka brokers.
It enables Kafka to read all files on Persistent Volumes before starting to avoid this issue.
You also can set this property during deployment using the custom environment variable property.
For example, `environment_variables: SCAN_FILE_SYSTEM=true`.

Reading files can take long time, from a few seconds to a few minutes.
If you have a big Kafka storage you also need to increase the liveness probe initial delay.

**Note**: The above workaround does not guarantee Kafka to work fine with NFS.
To avoid side effects, it is recommended to plan the transition to local storage.

### Recommendations

Not applicable

## Kafka Pod Is Not Ready With java.lang.InternalError

### Description

If one of Kafka pods is not ready for a long time, and logs have many errors like the following:

```text
[2021-08-13T13:18:44,416][ERROR][category=kafka.server.ReplicaManager] [ReplicaManager broker=2] Error processing fetch with max size 10000000 from replica [1] on partition connect-configs-0: (fetchOffset=763036, logStartOffset=0, maxBytes=10000000, currentLeaderEpoch=Optional[26])
java.lang.InternalError: a fault occurred in a recent unsafe memory access operation in compiled Java code
	at kafka.log.OffsetIndex$$Lambda$295/0x00000001003eec40.get$Lambda(Unknown Source)
	at kafka.log.OffsetIndex.lookup(OffsetIndex.scala:89)
```

It means there are some issues with file system, memory or JVM. There are a lot of causes lead to this issue.
Some information about causes described in [stackoverflow answer](https://stackoverflow.com/a/45536678) or
ticket [KAFKA-5628](https://issues.apache.org/jira/browse/KAFKA-5628).

### Alerts

Not applicable

### Stack trace

```text
[2021-08-13T13:18:44,416][ERROR][category=kafka.server.ReplicaManager] [ReplicaManager broker=2] Error processing fetch with max size 10000000 from replica [1] on partition connect-configs-0: (fetchOffset=763036, logStartOffset=0, maxBytes=10000000, currentLeaderEpoch=Optional[26])
java.lang.InternalError: a fault occurred in a recent unsafe memory access operation in compiled Java code
	at kafka.log.OffsetIndex$$Lambda$295/0x00000001003eec40.get$Lambda(Unknown Source)
	at kafka.log.OffsetIndex.lookup(OffsetIndex.scala:89)
```

### How to solve

To resolve this issue, a workaround can be used. You need to restart problem Kafka pods.

### Recommendations

To investigate this issue you need to check are there some issues on disk storage and make sure storage has enough space.

It is necessary to verify memory configurations for your Kafka brokers,
it is recommended to set HEAP_SIZE not more than 3/4 of available memory.

It is also recommended to set `memory.requests` = `memory.limits` for Kafka.

## Topics with Insufficient Replication Factor

### Description

Kafka has a lot of the following error:

```yaml
2022-03-22T06:58:20,521][ERROR][category=kafka.server.ReplicaManager] [ReplicaManager broker=2] Error processing append operation on partition topic-0 org.apache.kafka.common.errors.NotEnoughReplicasException: The size of the current ISR Set(1) is insufficient to satisfy the min.isr requirement of 2 for partition topic-0
```

This could mean that you have a topic with ReplicationFactor less than the value of min in-sync replicas for the topic.

Most times it happens for topics created by microservices with replication factor = 1 in Kafka clusters with 3 and more brokers.
This is an incorrect configuration and could lead to error with producing data with `acks=all` and other unexpected issues like
`UNKNOWN_LEADER_EPOCH`.

To check the replication factor, you can use the command:

```bash
./bin/kafka-topics.sh --bootstrap-server localhost:9092 --describe --topic topic --command-config bin/adminclient.properties
```

### Alerts

* [KafkaPartitionCountAlert](./alerts.md#kafkapartitioncountalert)

### Stack trace

```yaml
2022-03-22T06:58:20,521][ERROR][category=kafka.server.ReplicaManager] [ReplicaManager broker=2] Error processing append operation on partition topic-0 org.apache.kafka.common.errors.NotEnoughReplicasException: The size of the current ISR Set(1) is insufficient to satisfy the min.isr requirement of 2 for partition topic-0
```

### How to solve

To resolve it, first find the microservice which creates incorrect topics and fix the issue on its side
(change replication factor for new topics).

To fix the existing topics, execute the following script:

```sh
./bin/kafka-partitions.sh rebalance_topic topic
```

Or the following script for all topics with an incorrect replication factor:

```sh
./bin/kafka-partitions.sh rebalance
```

Or delete the topics and allow the microservice to create it again with correct configurations.

**Important**: This script does not work correctly for versions of docker-kafka images less than 2.8.1-0.23 and can lead to the issue,
[Kafka CPU is overloaded only for one of the cluster nodes](#kafka-cpu-is-overloaded-only-for-one-of-the-cluster-nodes).

**Pay attention**, this method could not work for some of topic partitions and you could still see ISR issue.
In that case there is no correct solution and you can fix it only with removing corrupted partitions from Kafka storage.

For example, for the error log:

```text
ISR Set(1) is insufficient to satisfy the min.isr requirement of 2 for partition topic-0
```

You can remove partition `topic-0` from the disk with the command:

```sh
rm -rf /var/opt/kafka/data/2/topic-0
```

Where `/2/` is number of Kafka pod where the command is executing.

**Pay attention**, performing this command can lead to data loss.

### Recommendations

Not applicable

## Kafka CPU is Overloaded only for one of the Cluster Nodes

### Description

There can be many causes why one of the node can be overloaded, but most often it is due to incorrect partition leaders distribution.

You can find partitions and leaders distribution on Kafka Monitoring dashboard  here, ([Monitoring](./monitoring.md)).

You can also describe topics with leaders using the command:

```bash
./bin/kafka-topics.sh --bootstrap-server localhost:9092 --describe --command-config bin/adminclient.properties
```

Example of incorrect leader distribution (all leaders on node 1):

```text
Topic: test_topic    PartitionCount: 10      ReplicationFactor: 3    Configs: min.insync.replicas=2,segment.bytes=1073741824,max.message.bytes=10000000
        Topic: test_topic    Partition: 0    Leader: 1       Replicas: 1,2,3 Isr: 3,1,2
        Topic: test_topic    Partition: 1    Leader: 1       Replicas: 1,2,3 Isr: 3,1,2
        Topic: test_topic    Partition: 2    Leader: 1       Replicas: 1,2,3 Isr: 3,1,2
        Topic: test_topic    Partition: 3    Leader: 1       Replicas: 1,2,3 Isr: 3,1,2
        Topic: test_topic    Partition: 4    Leader: 1       Replicas: 1,2,3 Isr: 3,1,2
```

### Alerts

* [KafkaCPUUsageAlert](./alerts.md#kafkacpuusagealert)

### Stack trace

Not applicable

### How to solve

To resolve it, use the following command:

```sh
./bin/kafka-partitions.sh rebalance_leaders_topic test_topic
```

or for all topics:

```sh
./bin/kafka-partitions.sh rebalance_leaders
```

**Important**: This command is available only for docker-kafka version more than 2.8.1-0.23.

### Recommendations

Not applicable

## Displaying Consumer Groups Fails with "This server does not host this topic-partition"

### Description

The AKHQ fails on loading topics or consumer groups with the following issue:

```text
java.lang.RuntimeException: Error for Describe Topics Offsets {}
	at org.akhq.utils.Logger.call(Logger.java:31)
	at org.akhq.modules.AbstractKafkaWrapper.describeTopicsOffsets(AbstractKafkaWrapper.java:121)
...
	at java.base/java.util.concurrent.ThreadPoolExecutor.runWorker(Unknown Source)
	at java.base/java.util.concurrent.ThreadPoolExecutor$Worker.run(Unknown Source)
	at java.base/java.lang.Thread.run(Unknown Source)
Caused by: org.apache.kafka.common.errors.UnknownTopicOrPartitionException: This server does not host this topic-partition.
```

In Kafka, you can also find the following logs:

```text
[2022-03-11T08:23:40,986][WARN][category=kafka.server.ReplicaFetcherThread] [ReplicaFetcher replicaId=4, leaderId=5, fetcherId=0] Received UNKNOWN_TOPIC_OR_PARTITION from the leader for partition cip-engine-libraries-cloudbss-kube-qa2-0. This error may be returned transiently when the partition is being created or deleted, but it is not expected to persist.
```

It specifies that some consumer groups failed on the describe operation and AKHQ (or Kafka CLI) failed on the bulk operation
to describe groups.
This can happen when some topics have been removed in an unexpected way.

To find the problem groups, use the command:

```bash
./bin/kafka-consumer-group-checker.sh check
```

It displays a list of failed consumer groups.

### Alerts

Not applicable

### Stack trace

```text
[2022-03-11T08:23:40,986][WARN][category=kafka.server.ReplicaFetcherThread] [ReplicaFetcher replicaId=4, leaderId=5, fetcherId=0] Received UNKNOWN_TOPIC_OR_PARTITION from the leader for partition cip-engine-libraries-cloudbss-kube-qa2-0. This error may be returned transiently when the partition is being created or deleted, but it is not expected to persist.
```

### How to solve

To resolve the issue:

1. Remove the problematic consumer group. Note that this also removes all consumer offsets for members of this group.
2. Find consumer group owners and create topics which are not existing currently.

### Recommendations

Not applicable

## Synchronization problem between Kafka and ZooKeeper cluster

### Description

ZooKeeper cluster re-installation or manual cleanup of topics znodes can cause problems in cluster functioning if data is produced
into affected Kafka topics at the same time.
If there is no data about topic in ZooKeeper, it is created with new `topicId` on next producer request in such topic.
But `partition.metadata` of such topic still contains old `topicId`.
It leads to error on both producer and consumer sides.

Producer side fails with following error:

```text
[2023-03-22T14:13:11,749][WARN][category=org.apache.kafka.clients.producer.internals.Sender] [Producer clientId=console-producer] Got error produce response with correlation id 6 on topic-partition test-9, retrying (2 attempts left). Error: UNKNOWN_TOPIC_OR_PARTITION
[2023-03-22T14:13:11,751][WARN][category=org.apache.kafka.clients.producer.internals.Sender] [Producer clientId=console-producer] Received unknown topic or partition error in produce request on partition test-9. The topic-partition may not exist or the user may not have Describe access to it
```

Consumer side fails with inconsistent topic ID error:

```text
[2023-03-22T14:15:25,348][WARN][category=org.apache.kafka.clients.consumer.internals.Fetcher][test-connector|task-0]  [Consumer clientId=connector-consumer-test-connector-0, groupId=connect-test-connector] Received inconsistent topic ID error in fetch for partition test-8
```

Also, such problem can be detected in `state-change.log` file:

```text
[2023-03-22T14:15:25,348]ERROR [Broker id=0] Topic Id in memory: jKTRaM_TSNqocJeQI2aYOQ does not match the topic Id for partition test-0 provided in the request: nI-JQtPwQwGiylMfm8k13w. (state.change.logger)
```

One more symptom of this issue is frozen replication between brokers and high CPU usage.
It can be checked with `describe` command, most part of topics have only one number in `Isr` column:

```sh
./bin/kafka-topics.sh --bootstrap-server localhost:9092 --describe --command-config bin/adminclient.properties

Topic: connect-offset       TopicId: p2uQ6DWeRBqmJVReJtTeYw PartitionCount: 25      ReplicationFactor: 3   Configs: min.insync.replicas=2,cleanup.policy=compact,segment.bytes=1073741824
        Topic: connect-offset       Partition: 0    Leader: 2       Replicas: 2,3,1 Isr: 2
        Topic: connect-offset       Partition: 1    Leader: 2       Replicas: 3,1,2 Isr: 2
Topic: streaming_service_metric TopicId: DGYL8aYxTKOuO2zAQos0BA PartitionCount: 5       ReplicationFactor: 3   Configs: min.insync.replicas=2,cleanup.policy=compact,segment.bytes=1073741824
        Topic: streaming_service_metric Partition: 0    Leader: 1       Replicas: 3,2,1 Isr: 1
        Topic: streaming_service_metric Partition: 1    Leader: 1       Replicas: 1,3,2 Isr: 1
```

Issue can be confirmed by comparing `topicId` in ZooKeeper configuration and topic `partition.metadata`:

```sh
./bin/kafka-topics.sh --bootstrap-server localhost:9092 --describe --command-config bin/adminclient.properties --topic test

Topic: test     TopicId: nI-JQtPwQwGiylMfm8k13w PartitionCount: 10      ReplicationFactor: 3    Configs: min.insync.replicas=2,segment.bytes=1073741824
        Topic: test     Partition: 0    Leader: 3       Replicas: 1,2,3 Isr: 1
        Topic: test     Partition: 1    Leader: 1       Replicas: 2,3,1 Isr: 1
        Topic: test     Partition: 2    Leader: 3       Replicas: 3,1,2 Isr: 1
        ...
```

```sh
cat /var/opt/kafka/data/1/test-0/partition.metadata
version: 0
topic_id: jKTRaM_TSNqocJeQI2aYOQ
```

### Alerts

Not applicable

### Stack trace

```text
[2023-03-22T14:15:25,348]ERROR [Broker id=0] Topic Id in memory: jKTRaM_TSNqocJeQI2aYOQ does not match the topic Id for partition test-0 provided in the request: nI-JQtPwQwGiylMfm8k13w. (state.change.logger)
```

### How to solve

1. Full data restore for ZooKeeper and Kafka storages.
2. Revert `topicId` in ZooKeeper configuration.
3. Replace `topicId` in Kafka `partition.metadata` with new ID from ZooKeeper.

**Pay attention**, only full data restore can guarantee topic data will be recovered fully and correctly.
Restoring `topicId` does not restore topic configurations.

### Recommendations

Not applicable

## Restore Storage for ZooKeeper and Kafka

### Description

The most accurate and reliable method for resolving this issue involves restoring both ZooKeeper and Kafka Persistent Volumes to
the latest available backup taken prior to the storage clearance.

Given that Product Kafka lacks inherent backup and restore mechanisms, achieving this resolution necessitates the utilization of backup
and restoration processes for the persistent volumes. One viable tool for this purpose is **Velero**, which facilitates the backup
and restoration of Kubernetes resources.

### Alerts

Not applicable

### Stack trace

Not applicable

### How to solve

The recommended procedure for executing this restoration process encompasses the following steps:

1. Scaling Down Kafka.

   To initiate the restoration process, it's advised to scale down the Kafka deployments.

2. Restoring ZooKeeper Storage.

   Begin by restoring the ZooKeeper storage from the previously captured backup.

3. Restoring Kafka Storage.

   Subsequently, proceed with the restoration of the Kafka storage utilizing the backup data.
   There should be Kafka snapshot taken for a near time to ZooKeeper snapshot.

4. Scaling Up Kafka.

   Conclude the restoration procedure by scaling up the Kafka deployments if they are no scaled by restore.

### Recommendations

Not applicable

## Dealing with Kafka's Under-replicated Partitions or Heaviest Partitions in Topics Due to Data Sync Issues

### Description

Kafka currently exhibits a topic partition distribution with async under replicated partitions with ReplicationFactor & ISR,
considerable consumption evidenced by the following examples:

Async under replicated partitions with ReplicationFactor & ISR:

```sh
/bin/kafka-topics.sh --bootstrap-server localhost:9092 --describe --topic test --command-config bin/adminclient.properties 

Topic: test     TopicId: nI-JQtPwQwGiylMfm8k13w PartitionCount: 10      ReplicationFactor: 3    Configs: min.insync.replicas=2,segment.bytes=1073741824
        Topic: test     Partition: 0    Leader: 3       Replicas: 1,2,3 Isr: 1
        Topic: test     Partition: 1    Leader: 1       Replicas: 2,3,1 Isr: 1,3
        Topic: test     Partition: 2    Leader: 3       Replicas: 3,1,2 Isr: 1
```

or,

Partition distribution data with considerable consumption:

```yaml
du -hsx $KAFKA_DATA_DIRS/$BROKER_ID/* | sort -rh | head -30

26G     /var/opt/kafka/data/1/__consumer_offsets-32
25G     /var/opt/kafka/data/1/__consumer_offsets-40
25G     /var/opt/kafka/data/1/__consumer_offsets-25
14G     /var/opt/kafka/data/1/__consumer_offsets-13
11G     /var/opt/kafka/data/1/__consumer_offsets-10
8.0G    /var/opt/kafka/data/1/__consumer_offsets-14
```

### Alerts

Not applicable

### Stack trace

Not applicable

### How to solve

To check the all replication partition distribution, you can use the command:

```bash
./bin/kafka-topics.sh --bootstrap-server localhost:9092 --describe --command-config bin/adminclient.properties 
``` 

Begin by collecting the data regarding the most heavily partitioned topics across all Kafka pods.
If noticeable, address the heaviest partitions topics in descending order by manual intervention:

To identify the top 5 highest partitions, execute the command:

```bash
./bin/kafka-partition-logs.sh --list --max-partitions 5
```  

To rectify the issue of high partition distribution, execute the following steps:

1. Access the Kafka pod.

2. Modify a topic by updating `cleanup.policy` and `retention.ms` using the command:

    ```bash
    ./bin/kafka-configs.sh --bootstrap-server localhost:9092 --command-config bin/adminclient.properties --entity-type topics --entity-name *<your_topic_name>* --alter --add-config cleanup.policy=delete
    
     ./bin/kafka-configs.sh --bootstrap-server localhost:9092 --command-config bin/adminclient.properties --entity-type topics --entity-name *<your_topic_name>* --alter --add-config retention.ms=3600000 
    ```

3. Restart kafka pods(Scale down and up all Kafka pods)

4. Revert the changes made to cleanup.policy and retention.ms properties for all altered Kafka topics using the following commands:

    ```sh
     ./bin/kafka-configs.sh --bootstrap-server localhost:9092 --command-config bin/adminclient.properties --entity-type topics --entity-name *<your_topic_name*> --alter --add-config cleanup.policy=compact
    
     ./bin/kafka-configs.sh --bootstrap-server localhost:9092 --command-config bin/adminclient.properties --entity-type topics --entity-name *<your_topic_name*> --alter --add-config retention.ms=604800000
    ```

   Alternatively, you can perform the same actions through the AKHQ UI. Search for the topic with the highest consumption and adjust
   the `cleanup.policy` and `retention.ms` values.

   **Note**: Ensure to create backups of the current `cleanup.policy` and `retention.ms` attributes to avoid data loss during
   the reversion process.

### Recommendations

Not applicable

## Kafka Pods Fail with "node already exists" Error

### Description

The following error in Kafka log means that ZooKeeper already has ephemeral ZNode for failed Kafka pod:

```text
[2021-09-07T07:49:02,767][ERROR][category=kafka.zk.KafkaZkClient$CheckedEphemeral] Error while creating ephemeral at /brokers/ids/1, node already exists and owner '72644107655650204' does not match current session '145773443801686401'
[2021-09-07T07:49:02,773][ERROR][category=kafka.server.KafkaServer] [KafkaServer id=1] Fatal error during KafkaServer startup. Prepare to shutdown
```

There can be 2 reason for this issue:

1. You try to install Kafka cluster with ZooKeeper which is already used for another Kafka cluster.
   To resolve this issue refer to [How to deploy several Kafka clusters with one ZooKeeper](/docs/public/installation.md#how-to-deploy-several-kafka-clusters-with-one-zookeeper).
2. There are Kafka pods in terminating status which block ZNode.
   To resolve this issue you can force delete terminating pods. For example:

   ```bash
   kubectl delete pod kafka-1-dsa7dsa7d77 --force -n kafka-namespace
   ```

### Alerts

Not applicable

### Stack trace

```text
[2021-09-07T07:49:02,767][ERROR][category=kafka.zk.KafkaZkClient$CheckedEphemeral] Error while creating ephemeral at /brokers/ids/1, node already exists and owner '72644107655650204' does not match current session '145773443801686401'
[2021-09-07T07:49:02,773][ERROR][category=kafka.server.KafkaServer] [KafkaServer id=1] Fatal error during KafkaServer startup. Prepare to shutdown
```

### How to solve

Pay attention, pods in terminating status could be a symptom of issues with Kubernetes nodes, report this issue to corresponding Cloud IT team.

### Recommendations

Not applicable

## Kafka Works Slowly and Consumes a lot of CPU For All Nodes

### Description

There are lot of symptoms of this issue like high CPU consumption for small incoming load, or increasing
the producer latency of consumer lag. One of additional symptoms of this issue is timeout exception on client sides.

The most usual cause of such behaviour is overloaded  Kafka with big topic partitions number.

There are strong restrictions for every Kafka cluster type and allowed number of partitions,
you can find then in [HWE](/docs/public/installation.md#hwe).

### Alerts

* [KafkaPartitionCountAlert](./alerts.md#kafkapartitioncountalert)

### Stack trace

Not applicable

### How to solve

You can check the number of partitions with the Grafana dashboard or performing the following command from any Kafka pod:

```sh
bin/kafka-topics.sh --bootstrap-server localhost:9092 --command-config bin/adminclient.properties --describe
```

If the number of partitions per broker exceeds the allowed number regarding HWE you need to decrease it to allowed using
the following approaches:

1. Remove unused topics (Kafka spends resource even on empty and not used topics).
   Pay attention, MaaS created topics should be removed only via MaaS.
2. Scale In Kafka cluster with new brokers using [Scaling Guide](/docs/public/scaling.md).
3. Spit one Kafka cluster into several for each project environment. This is most recommended option,
   because it covers the real production scenario.

### Recommendations

Not applicable

## Kafka Unhealthy After Cluster Scale-In

### Description

Kafka does not support out-of-box scaling-in process by simple decreasing number of brokers.

For example, if you have 5 brokers and scale in the cluster to 3 brokers without required data distribution before brokers deletion,
when describing topics you will see the following situation:

Command to describe all Kafka topics:

```yaml
./bin/kafka-topics.sh --bootstrap-server localhost:9092 --describe --command-config bin/adminclient.properties
```

```text
Topic: test.topic	TopicId: FJfeUNv5TWWLU98FWg89qg	PartitionCount: 5	ReplicationFactor: 3	Configs: min.insync.replicas=2,segment.bytes=1073741824
	Topic: test.topic	Partition: 0	Leader: none	Replicas: 4,5,1	Isr: 5
	Topic: test.topic	Partition: 1	Leader: none	Replicas: 5,1,2	Isr: 5
	Topic: test.topic	Partition: 2	Leader: 3	    Replicas: 1,2,3	Isr: 1,2,3
	Topic: test.topic	Partition: 3	Leader: 1	    Replicas: 2,1,3	Isr: 2,1,3
	Topic: test.topic	Partition: 4	Leader: none	Replicas: 3,4,5	Isr: 5
```

Since these topic partitions have replicas on 4 and 5 brokers, they will not be able to work properly.

### Alerts

Not applicable

### Stack trace

Not applicable

### How to solve

To solve this issue, follow the next steps:

1. Set `unclean.leader.election.enable=true` property in Kafka custom resource (CR).

    ```yaml
      kafka:
        environmentVariables:
          - CONF_KAFKA_UNCLEAN_LEADER_ELECTION_ENABLE=true
    ```

   **Note**: This won't work in case you set this parameter directly on topic.
   You need to change this parameter on each problematic topic as in the following example:

    ```sh
    ./bin/kafka-configs.sh --bootstrap-server localhost:9092 --command-config bin/adminclient.properties --entity-type topics --entity-name *<your_topic_name>* --alter --add-config unclean.leader.election.enable=true
    ```

2. Run the script to rebalance leaders:

    ```sh
    ./bin/kafka-partitions.sh rebalance_leaders
    ```

3. Check if all topics are working fine via topic describe command or AKHQ:

    ```sh
    ./bin/kafka-topics.sh --bootstrap-server localhost:9092 --describe --command-config bin/adminclient.properties
    ```

4. Delete `unclean.leader.election.enable=true` from CR, that will make Kafka brokers to restart without this property,
   it will force the Kafka brokers to restart with the updated configuration.

**Important**: Do not forget to remove the property from the topics for which you have specified it directly.

In case Kafka is working fine after the actions taken, but you are unable to create topics with following error:

```text
Topic `test-topic` is marked for deletion
```

Problem still persist on Zookeeper side, you should follow next steps to resolve this issue:

1. Configure the Zookeeper admin client, refer to [ZooKeeper Secure Client](https://github.com/Netcracker/qubership-zookeeper/tree/main/docs/public/security.md#zookeeper-clients-security-properties)

2. Check that topic, which you are unable to create, is in to-delete topics node:

    ```sh
    ls /admin/delete_topics
    ```

3. If there are problematic topics in the output, then you have to delete them by the following command:

    ```sh
    deleteall /admin/delete_topics
    ```

   After deletion, recreate `delete_topics node`:

    ```sh
    create /admin/delete_topics
    ```

   **Note**: If `ls /admin/delete_topics` command output contains working topics, you should delete only problematic ones using following
   command for each topic:

    ```sh
    deleteall /admin/delete_topics/<your_topic_name>
    ```

4. Go to Kafka and check if there are still broker partitions of problematic topics. Check for all partitions by the following command:

    ```sh
    ls /var/opt/kafka/data/<BROKER_ID>
    ```

   Delete all partitions which related to problematic topic

    ```sh
    rm -rf /var/opt/kafka/data/<BROKER_ID>/test-topic-0
    rm -rf /var/opt/kafka/data/<BROKER_ID>/test-topic-1
    ...
    ```

   **Important**: It is mandatory to delete broken topics partitions from ALL brokers in the cluster.

5. Repeat steps with topic leaders rebalance.

### Recommendations

Not applicable

## Container Failed with Error: container has runAsNonRoot and image will run as root

### Description

The Operator deploys successfully and operator logs do not contain errors, but Kafka Monitoring pod fails with the following error:

```text
Error: container has runAsNonRoot and image will run as root
```

Kafka Monitoring does not have special user to run processes, so default (`root`) user is used.
If you miss the `securityContext` parameter in the pod configuration and `Pod Security Policy` is enabled,
the default `securityContext` for pod is taken from `Pod Security Policy`.

If you configure the `Pod Security Policy` as follows then the error mentioned above occurs:

```yaml
runAsUser:
  # Require the container to run without root privileges.
  rule: 'MustRunAsNonRoot'
```

### Alerts

Not applicable

### Stack trace

```text
Error: container has runAsNonRoot and image will run as root
```

### How to solve

You need to specify the correct `securityContext` in pod configuration during installation.
For example, for Kafka Monitoring you should specify the following parameter:

```yaml
securityContext:
    runAsUser: 1000
```

### Recommendations

Not applicable

## Pod Evicted with Error: The node was low on resource: ephemeral-storage

### Description

If you encounter the problem that the Kafka pod has "Evicted" status due to the following error:

```text
The node was low on resource: ephemeral-storage. Container kafka was using 268980Ki, which exceeds its request of 0.
```

### Alerts

Not applicable

### Stack trace

Not applicable

### How to solve

You should set a quota `limits.ephemeral-storage`, `requests.ephemeral-storage` to limit this, as
otherwise any container can write any amount of storage to its node filesystem. For example:

```yaml
resources:
  limits:
    cpu: '2'
    ephemeral-storage: 1Gi
    memory: 3Gi
  requests:
    cpu: 700m
    ephemeral-storage: 1Gi
    memory: 1500Mi
```

### Recommendations

Not applicable

## CRD Creation Failed on OpenShift 3.11

### Description

If Helm deployment or manual application of CRD failed with the following error,
it depicts that the Kubernetes version is 1.11 (or less) and it is not compatible with the new format of CRD:

```text
The CustomResourceDefinition "kmmconfigs.qubership.org" is invalid: spec.validation.openAPIV3Schema: Invalid value:....
: must only have "properties", "required" or "description" at the root if the status subresource is enabled
```

For more information, refer to [https://github.com/jetstack/cert-manager/issues/2200](https://github.com/jetstack/cert-manager/issues/2200).

### Alerts

Not applicable

### Stack trace

Not applicable

### How to solve

To fix the issue, you need to find the following section in the CRD (`config/crd/old/qubership.org_kmmconfigs.yaml`):

```yaml
# Comment it if you deploy to Kubernetes 1.11 (e.g OpenShift 3.11)
type: object
```

Comment or delete row `type: object`, and then apply the CRD manually.

**Note**: You need to disable CRD creation during installation in case of such errors.

### Recommendations

Not applicable
