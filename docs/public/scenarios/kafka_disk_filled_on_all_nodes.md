The section describes the problem detection techniques for issues with filled disks.

# Metric

You can use information about mounted volumes that is stored in `t_disk` measurement with fields:

- `used`
- `used_percent`

To retrieve information about how much disk space Kafka is using:

- Run the command below inside Kafka container.

  ```sh
  df -h /var/opt/kafka/data
  ```

Possible output:

```text
Filesystem      Size  Used Avail Use% Mounted on                                                                                                                                                       
/dev/vdc        2.0G  6.5M  1.8G   1% /var/opt/kafka/data
```

# Grafana Dashboard

To retrieve the metric of disk usage the following query can be used:

```sql
SELECT mean("used_percent") FROM "t_disk" WHERE ("path" =~ /.*$broker$/) AND $timeFilter GROUP BY time($__interval), "path" fill(null)
```

# Logging

When the problem occurs, you can see the following exception in the console logs:

```text
[2018-03-16T13:57:35,035][FATAL][category=kafka.server.ReplicaManager] [Replica Manager on Broker 1]: Halting due to unrecoverable I/O error while handling produce request: 
kafka.common.KafkaStorageException: I/O exception in append to log 'disk_filled_on_all_nodes-test-topic-0'
	at kafka.log.Log.append(Log.scala:585)
	at kafka.log.Log.appendAsLeader(Log.scala:549)
	at kafka.cluster.Partition$$anonfun$10.apply(Partition.scala:486)
	at kafka.cluster.Partition$$anonfun$10.apply(Partition.scala:474)
	at kafka.utils.CoreUtils$.inLock(CoreUtils.scala:213)
	at kafka.utils.CoreUtils$.inReadLock(CoreUtils.scala:219)
	at kafka.cluster.Partition.appendRecordsToLeader(Partition.scala:473)
	at kafka.server.ReplicaManager$$anonfun$appendToLocalLog$2.apply(ReplicaManager.scala:546)
	at kafka.server.ReplicaManager$$anonfun$appendToLocalLog$2.apply(ReplicaManager.scala:532)
	at scala.collection.TraversableLike$$anonfun$map$1.apply(TraversableLike.scala:234)
	at scala.collection.TraversableLike$$anonfun$map$1.apply(TraversableLike.scala:234)
	at scala.collection.Iterator$class.foreach(Iterator.scala:891)
	at scala.collection.AbstractIterator.foreach(Iterator.scala:1334)
	at scala.collection.IterableLike$class.foreach(IterableLike.scala:72)
	at scala.collection.AbstractIterable.foreach(Iterable.scala:54)
	at scala.collection.TraversableLike$class.map(TraversableLike.scala:234)
	at scala.collection.AbstractTraversable.map(Traversable.scala:104)
	at kafka.server.ReplicaManager.appendToLocalLog(ReplicaManager.scala:532)
	at kafka.server.ReplicaManager.appendRecords(ReplicaManager.scala:373)
	at kafka.server.KafkaApis.handleProduceRequest(KafkaApis.scala:462)
	at kafka.server.KafkaApis.handle(KafkaApis.scala:97)
	at kafka.server.KafkaRequestHandler.run(KafkaRequestHandler.scala:66)
	at java.lang.Thread.run(Thread.java:745)
Caused by: java.io.IOException: No space left on device
	at sun.nio.ch.FileDispatcherImpl.write0(Native Method)
	at sun.nio.ch.FileDispatcherImpl.write(FileDispatcherImpl.java:60)
	at sun.nio.ch.IOUtil.writeFromNativeBuffer(IOUtil.java:93)
	at sun.nio.ch.IOUtil.write(IOUtil.java:65)
	at sun.nio.ch.FileChannelImpl.write(FileChannelImpl.java:211)
	at org.apache.kafka.common.record.MemoryRecords.writeFullyTo(MemoryRecords.java:91)
	at org.apache.kafka.common.record.FileRecords.append(FileRecords.java:153)
	at kafka.log.LogSegment.append(LogSegment.scala:122)
	at kafka.log.Log.append(Log.scala:668)
	... 22 more
```

# Retention Policy

Kafka uses log data structure to manage its messages. Log data structure is basically an ordered 
set of segments whereas a segment is a collection of messages. Kafka provides retention at segment 
level instead of at message level. Hence, Kafka keeps on removing segments from its end as these 
violate retention policies.

For more information, see [Broker Configs](https://kafka.apache.org/documentation.html#brokerconfigs).

Looking at the broker disk, each topic partition is a directory containing the corresponding segment files and other 
files. The name of the segment file defines the starting offset of the records in that log. The segment file with 
the latest offset is active segment and is the only file open for read and write operations.
The following is an example of what the directory looks like.

```sh
$ ls -l __consumer_offsets-7
total 76692
-rw-rw-r--    1 kafka    kafka            0 Jan  2 15:10 00000000000000000000.index
-rw-rw-r--    1 kafka    kafka          623 Jan  2 15:10 00000000000000000000.log
-rw-rw-r--    1 kafka    kafka           12 Jan  2 15:10 00000000000000000000.timeindex
-rw-rw-r--    1 kafka    kafka            0 Jan  4 04:49 00000000000001022104.index
-rw-rw-r--    1 kafka    kafka          206 Jan  4 04:49 00000000000001022104.log
-rw-rw-r--    1 kafka    kafka           10 Jan  2 15:10 00000000000001022104.snapshot
-rw-rw-r--    1 kafka    kafka           12 Jan  4 04:49 00000000000001022104.timeindex
-rw-rw-r--    1 kafka    kafka     10485760 Jan 10 14:15 00000000000001531121.index
-rw-rw-r--    1 kafka    kafka     78099919 Jan 10 14:15 00000000000001531121.log
-rw-rw-r--    1 kafka    kafka           10 Jan  4 04:49 00000000000001531121.snapshot
-rw-rw-r--    1 kafka    kafka     10485756 Jan 10 14:15 00000000000001531121.timeindex
-rw-r--r--    1 kafka    kafka           10 Jan 10 11:17 00000000000001865362.snapshot
-rw-r--r--    1 kafka    kafka           38 Jan 10 11:18 leader-epoch-checkpoint
```

From the output, you can see that the first log segment `00000000000000000000.log` contains records from offset 0 to 
offset 1022104 and the second segment `00000000000001022104.log` contains records starting from offset 1022104. These 
files are called the **passive segments**. The third segment `00000000000001531121.log` contains records starting from 
offset 1531121 and is called the **active segment**.

## Time based Retention Policy

Under this policy, we configure the maximum time a segment (hence messages) can live for. Once 
a segment has spanned configured retention time, it is marked for deletion. Default retention time 
for segments is 7 days:

```text
log.retention.ms=604800000
log.retention.minutes=10080
log.retention.hours=168
```

Kafka deletes only the passive log segments. You have to tune either `log.roll.ms` or `log.roll.hours` to roll 
the active log segment into passive one:

```text
log.roll.ms=604800000
log.roll.hours=168
```

These settings control the period of time after which Kafka will force the log to roll, even if the segment file is not 
full. This ensures that the retention process is able to delete or compact old data.

## Size based Retention Policy

In this policy, we configure the maximum size of data to retain in the log for each topic partition. Once log size 
reaches this size, it starts removing segments from its end. By default, this policy isn't enabled:

```text
#log.retention.bytes=1073741824
```

Kafka deletes only the passive log segments. You have to tune `log.segment.bytes` to roll the active log segment into 
passive one:

```text
log.segment.bytes=1073741824
```

This setting configures the maximum size in bytes of a log segment before an active segment is rolled. This ensures 
that the retention process is able to delete or compact old data.

Larger sizes mean the active segment contains more messages and is rolled less often. Non-active segments also become 
eligible for deletion less often.

# Troubleshooting Procedure

Because Kafka persists all data to disk, it is necessary to monitor the amount of free disk space available to Kafka. 
We recommend changing the retention policy so that disk usage does not exceed 85%.

To define the retention policy so that disk capacity doesn't exceed 85%, do the following:

1. Edit environment variables of Kafka deployment configuration.
2. Complete any of the following actions to set the retention policy to meet your needs:
	- Set either `CONF_KAFKA_LOG_RETENTION_MS`, `CONF_KAFKA_LOG_RETENTION_MINUTES` or `CONF_KAFKA_LOG_RETENTION_HOURS` environment
      variables to a value suitable for your use cases.<br>**Note** that either `CONF_KAFKA_LOG_ROLL_MS` or `CONF_KAFKA_LOG_ROLL_HOURS` 
      environment variables should have a value that less than or equal to **the retention time**.

	- Set `CONF_KAFKA_LOG_RETENTION_BYTES` environment variable to a value suitable for your use cases.<br>**Note** that
     `CONF_KAFKA_LOG_SEGMENT_BYTES` environment variable should have a value that less than or equal to the maximum size of the log
     (`CONF_KAFKA_LOG_RETENTION_BYTES` environment variable) before deleting it.<br>It is also important to note that this is the limit
     for each partition, so multiply this value by the number of partitions to calculate the total data retained for the topic.
3. Save your changes.

Continue to monitor the disk and make additional changes if it begins to exceed 85% disk capacity.

**Important**: If the disk size has been exceeded, then Kafka can be restored to work only after manually deleting the passive 
log segments.

# Cleanup Partition Logs By Retention

**Important**: We do not recommend to use it in case you have `compact` topics in which data shouldn't be erased.

In case Kafka doesn't start up due to full disk, you can clean partition logs from specific topics by `kafka-partition-logs.sh` script.

The script has three main commands with appropriate options: 

The `--list` command lists the heaviest partitions in descending order.

 - The `--max-partitions` parameter specifies the number of listed partitions. The parameter is optional, the default value is `10`.
   To list all partitions use `-1` value.

The `--delete` command deletes data from topic/partition by retention.

 - The `--topic` parameter specifies topic for which all partitions logs should be cleaned by retention.

 - The `--partition` parameter specifies partition number for appropriate topic to be cleaned.
   The parameter can only be used with `--topic` parameter.

 - The `--all` parameter is used to clean all topics.

 - The `--retention` parameter specifies retention time in hours after which logs can be deleted.
   The parameter is optional, the default value is `24`.

The `--help` command shows appropriate options for script.

Examples:

- List top 5 largest partitions.

```bash
./bin/kafka-partition-logs.sh --list --max-partitions 5 
```

- Clean logs from test_topic in particular partition `4` which were updated more than 48 hours ago.

```bash
./bin/kafka-partition-logs.sh --delete --topic test_topic --partition 4 --retention 48
```
