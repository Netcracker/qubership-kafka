This section describes the problem detection techniques for issues with memory limit.

# Metric

You can use information about used memory that is stored in `memory/usage` measurement in field
`value` and information about memory limit that is stored in `memory/limit` measurement in field
`value`.

These metrics show pod memory usage and limit in bytes. Constant high memory usage that is close
to memory limit may indicate a critical situation that service is overloaded or resource limits
are too low. It can potentially lead to the increase of response times or crashes.

# Grafana Dashboard

![Heap Usage](/docs/public/images/heap-usage.png)

To retrieve the metric of memory usage the following query can be used:

```sql
SELECT mean("value") FROM "memory/usage" WHERE ("pod_name" =~ /^$broker.*/ AND "type" = 'pod') AND $timeFilter GROUP BY time($__interval), "pod_name" fill(null)
```

To retrieve the metric of memory limit the following query can be used:

```sql
SELECT mean("value") FROM "memory/limit" WHERE ("pod_name" =~ /^$broker.*/ AND "type" = 'pod') AND $timeFilter GROUP BY time($__interval), "pod_name" fill(null)
```

# Logging

When the problem occurs, you can see the following exception in the console logs:

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

# Troubleshooting Procedure

If you see a high memory usage, you can either increase memory limit along with heap size or scale 
out the cluster by adding more nodes.

You can change heap size using `HEAP_OPTS` environment variable on deployment configuration.
