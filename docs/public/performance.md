The following topics are covered in this chapter:

[[_TOC_]]

# General Information

The section provides information about Kafka Cluster Performance Optimization.

# Prerequisites and limitations

Firstly to understand kafka cluster possibilities it's necessary to evaluate disk bandwidth of Kafka cluster. To check disk RW rate
the next command can be applied from Kafka pod:

```text
kafka-1:/opt/kafka$ dd if=/dev/zero of=/var/opt/kafka/data/test bs=1M count=65536 oflag=direct
65536+0 records in
65536+0 records out
68719476736 bytes (69 GB, 64 GiB) copied, 318.223 s, 216 MB/s
```

Need to check disks of all Kafka nodes to get the minimal value. If at least one the disks works slower than others
it can cause problems with performance during brokers synchronization.

If Kafka writes with rate close to returned by command above, it can not write faster (refer to Kafka Monitoring dashboard, Disk panel).
Otherwise, either Kafka cluster or Kafka clients can not produce enough much data to write with expected disk speed.
The reason of such issue on Kafka cluster side can be insufficient CPU or RAM resources provided for brokers.
To utilize CPU or RAM more efficiently refer to configurations below depending on required optimization goals.

# Optimization Goals

Depending on Kafka cluster and Kafka clients specifics different optimization goals can be achieved.
Some configuration parameters should be applied on Kafka Cluster startup, so it's better to consider
optimization steps before installation or cluster scaling.

## Optimizing for Throughput

To achieve high throughput, Kafka producers, broker and consumers need to process as much data as possible within a given amount of time. 

Such approach can be achieved by parallelism in Kafka and higher number of topic partitions.
Generally, the higher number of partitions, the higher throughput. But it can cause higher latency and unavailability.

Other point is the batch size producer can send together in a single request with `batch.size` parameter.
Larger batch sizes result in fewer requests to the brokers and lower CPU overhead on each request processing.
To give more time for batches to fill, you can configure the `linger.ms` parameter to have the producer wait longer before sending,
but note that it messages are not sent as soon as they are ready. By default, `batch.size=16384` and `linger.ms=0`.

Similar approach can be applied for consumers with increased `fetch.min.bytes` parameter.
Higher `fetch.min.bytes` reduces the number of fetch requests, reducing also CPU overhead,
but increasing time until message can be consumed.

Applying compression can also improve throughput. For higher performance, compression types `lz4` or `gzip` are recommended.
Compression is applied on full batches of data, so the `batch.size` can also affect the efficiency of such improvement.
Compression can be enabled with `compression.type` parameter, but it also causes higher CPU usage. 

When a producer sends a message to a Kafka cluster it awaits for a confirmation from the leader before proceeding to send next messages.
The number of acknowledgments the leader broker must receive before response to the producer can be configured with `acks` property.
If `acks=all` producer waits until the message is replicated to other brokers.
In opposite if `acks=1` then leader broker writes the record locally and sends response to producer without additional confirmation.
It can improve throughput but causes lower durability.

Network threads handle requests to the Kafka cluster, such as produce and fetch requests from client applications.
Adjust the number of network threads (`num.network.threads`) to reflect the replication factor and the levels of activity from client
producers and consumers interacting with the Kafka cluster.

I/O threads pick up requests from the request queue,
so adding more threads to process data can improve throughput at the cost of higher CPU usage.
To increase the number of threads adjust the `num.io.threads` parameter.

To improve throughput problems caused by number of disks limitation network buffer size can be extended.
The `replica.socket.receive.buffer.bytes` defines the size of the buffer for network requests.
According to buffer size the maximum number of bytes Kafka cluster can receive defined by `socket.request.max.bytes` parameter.

If message batches size is modified buffers for sending and receiving messages might be insufficient for the required throughput.
Buffer sizes can be defined with `socket.send.buffer.bytes` and `socket.receive.buffer.bytes` parameters.

Example of configuration for higher throughput:

```yaml
kafka:
...
  environmentVariables:
    - CONF_KAFKA_NUM_IO_THREADS=16
    - CONF_KAFKA_NUM_NETWORK_THREADS=8
    - CONF_KAFKA_NUM_REPLICA_FETCHERS=4
    - CONF_KAFKA_REPLICA_SOCKET_RECEIVE_BUFFER_BYTES=1310720
    - CONF_KAFKA_SOCKET_RECEIVE_BUFFER_BYTES=2048000
    - CONF_KAFKA_SOCKET_SEND_BUFFER_BYTES=2048000
```

Producer:

```yaml
producerProps:
  - batch.size=1000000
  - linger.ms=100
  - compression.type=lz4
```

Consumer:

```yaml
fetchSize: 1048576
```

## Optimizing for Latency

Latency optimization implies lower time until the message can be processed by producer, broker and consumer.
Mostly, default parameters of Kafka cluster are set for lower latency goal.

Unlike throughput optimization optimizing for latency requires lower number of partitions.
A broker by default uses a single thread to replicate data from another broker,
so it may take longer to replicate a lot of partitions shared between each pair of brokers and consequently take longer for messages
to be considered committed.
It can be achieved by limiting number of partitions per broker or by increasing number of brokers.

To increase the degree of I/O parallelism in the follower broker, adjust the configuration parameter `num.replica.fetchers`,
which is the number of fetcher threads per source broker.
Start your benchmark tests at its default value of 1, and then increase it if followers can’t keep up with the leader.

Producers sends batches to brokers after delay defined in `linger.ms` parameter.
By default, `linger.ms=0`, so producer can send as soon as it got some message even if batch consists of only one message.
If low latency is needed, make sure this parameter left with default value.

Consumers in this case should also answer as soon as any data provided. So `fetch.min.bytes` should be set to `1` as default. 

Data compression can also cause additional processing time, so consider necessity of compression for producers as well.
Data compression is also disabled for broker by default (`compression.type=none`).

The number of producer acknowledgments affects the time on message processing.
The sooner the leader broker responds, the sooner producer can send the next message.
By default, `acks=1`, so the leader broker responds immediately after confirmation on its side without waiting for replicas.
Depending on requirements, `acks=0` can be specified which means that producer proceeds to the next message without any confirmation 
from brokers, but it significantly decreases durability because messages can potentially be lost without producer even knowing.

Example of configuration for lower latency:

```yaml
kafka:
...
  environmentVariables:
    - CONF_KAFKA_NUM_REPLICA_FETCHERS=4
```

Producer:

```yaml
producerProps:
  - linger.ms=0(default)
  - compression.type=none(default)
  - acks=1(default)
```

Consumer:

```yaml
fetchSize: 1(default)
```

## Optimizing for Durability

The main target of optimizing for durability is to reduce the chance for a message to get lost.
This goal can be achieved by data replication. 

Topics with high durability requirements should have the configuration parameter `replication.factor` set to `3`.
If the topic auto creation mechanism is enabled (`auto.create.topics.enable=true`),
then it's recommended to apply `default.replication.factor=3` as well or disable auto topic creation to make sure all topics have necessary
replication factor. The configuration parameter `min.insync.replicas` specifies the minimum threshold for the replica count in the ISR list.

On producer side durability can be achieved by enabling all acknowledgements (`acks=all`).
In this case no new messages can be processed by producer until all replicas confirm the message committed.
It causes higher latency in Kafka cluster.

Also, producers can increase durability via retry mechanism.
The producer automatically tries to resend messages up to the number of times specified in `retries` parameter (default `MAX_INT`) and
up to the time specified in `delivery.timeout.ms` (default 120000).

Note that automatic retries can cause duplication and ordering problems in cluster.
It can be handled by idempotent producer (`enable.idempotence=true`) that organizes retries so broker ignores duplicate sequence number.
Otherwise, limiting of in-flight requests can also preserve message ordering.
If parameter `max.in.flight.requests.per.connection=1` message ordering guaranteed, but it lowers total performance.

In case of a broker failure, the Kafka cluster can automatically detect the failures and elect new partition leaders.
New partition leaders are chosen from the existing replicas that are running.
For higher durability, this should be disabled by setting `unclean.leader.election.enable=false` to ensure that new leaders
should be elected from just the ISR list. This prevents the chance of losing messages that were committed but not replicated.

Example of configuration for lower latency:

```yaml
kafka:
...
  environmentVariables:
    - CONF_KAFKA_DEFAULT_REPLICATION_FACTOR=3(default)
    - CONF_KAFKA_AUTO_CREATE_TOPICS_ENABLE=false
    - CONF_KAFKA_MIN_INSYNC_REPLICAS=2(default)
    - CONF_KAFKA_UNCLEAN_LEADER_ELECTION_ENABLE=false
```

Producer:

```yaml
producerProps:
  - enable.idempotence=true
  - acks=all
```

or

```yaml
producerProps:
  - max.in.flight.requests.per.connection=1
  - acks=all
```

## Optimizing for Availability

Availability optimization implies quick recovery after failure scenarios.

Higher number of partitions may increase throughput, but having more partitions can also increase recovery time in the event of
a broker failure. So the partitions number should be also considered.

All acknowledgments enabling (`acks=all`) with `min.insync.replicas` specified may lead to errors if such minimum cannot be met.
Setting `min.insync.replicas=1` let the system will tolerate more replica failures.
As long as the minimum number of replicas is met, the producer requests will continue to succeed,
which increases availability for the partition.

Broker failures may cause partition leader election.
In opposite to durability goal, unclean leader election is acceptable to improve availability
by setting `unclean.leader.election.enable=true`.

On consumer side there is default session timeout for interacting with broker (`session.timeout.ms`).
Lowering such timeout can help to detect failed consumer faster so recovery can be performed.
Set this as low as possible to detect hard failures but not so low that soft failures occur.
Default value for this timeout is `10000`.

Example of configuration for lower latency:

```yaml
kafka:
...
  environmentVariables:
    - CONF_KAFKA_MIN_INSYNC_REPLICAS=1
    - CONF_KAFKA_UNCLEAN_LEADER_ELECTION_ENABLE=true
```

Consumer:

```yaml
timeout:1000
```
