This section provides information about the SVT performed for Kafka Service.

## AWS Environment

EKS Nodes:

* [t3.xlarge](https://aws.amazon.com/ru/ec2/instance-types/t3/)

### Kafka Hardware

```yaml
replicas: 3
heapSize: 2048
resources:
  requests:
    cpu: 2
    memory: 12Gi
  limits:
    cpu: 4
    memory: 12Gi
    
Disk rate: 216 MB/s
```

### Producers Setup

```yaml
replicas: 5
throughput: 50000
numRecords: 20000000
recordSize: 1024
producerProps:
  - batch.size=1000000
  - linger.ms=10
```

### Consumers Setup

```yaml
consumerGroups: ["kafka-svt-1", "kafka-svt-2", "kafka-svt-3"]
replicas: 1
fetchSize: 1048576
socketBufferSize: 2097152
timeout: 60000
messages: 20000000
```

| Scenario                                       | Kafka Configs                                                                                                                                                                                                                                                                                           | Results                                                       |
|------------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------------------------------------------|
| 100 partitions                                 | Default                                                                                                                                                                                                                                                                                                 | Avg. Write Bytes: 197.21 MiB/S                                |
| 100 partitions<br>Enhanced Kafka Configuration | environmentVariables:<br>- CONF_KAFKA_NUM_IO_THREADS=32<br>- CONF_KAFKA_NUM_NETWORK_THREADS=12<br>- CONF_KAFKA_NUM_REPLICA_FETCHERS=12<br>- CONF_KAFKA_REPLICA_SOCKET_RECEIVE_BUFFER_BYTES=1310720<br>- CONF_KAFKA_SOCKET_RECEIVE_BUFFER_BYTES=2048000<br>- CONF_KAFKA_SOCKET_SEND_BUFFER_BYTES=2048000 | Avg. Write Bytes: 215.95 MiB/S                                |
| 5 Partitions                                   | Default                                                                                                                                                                                                                                                                                                 | Avg. Write Bytes: 148.7 MiB/S                                 |
| 5 partitions<br>acks=1                         | Default                                                                                                                                                                                                                                                                                                 | Write Bytes per Broker: 195.1 MiB/S, 139.8 MiB/S, 194.8 MiB/S |

## Kubernetes Environment Large HWE

Kubernetes Nodes:

* CPU: 2 x Intel Xeon Gold 6258R 2.7G, 28 C/56T
* Memory: 8 x DDR4-3200 64GB 2RX4
* Disk: Write rate 229 MB/s

### Kafka Hardware

```yaml
replicas: 3
heapSize: 2048
resources:
  requests:
    cpu: 1
    memory: 8Gi
  limits:
    cpu: 2
    memory: 16Gi
```

### Producers Setup

```yaml
replicas: 5
throughput: 50000
numRecords: 20000000
recordSize: 1024
producerProps:
  - batch.size=1000000
  - linger.ms=10
```

### Consumers Setup

```yaml
consumerGroups: ["kafka-svt-1", "kafka-svt-2", "kafka-svt-3"]
replicas: 1
fetchSize: 1048576
socketBufferSize: 2097152
timeout: 60000
messages: 20000000
```

| Scenario                                       | Kafka Configs                                                                                                                                                                                                                                                                                           | Results                                                         |
|------------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-----------------------------------------------------------------|
| 100 partitions                                 | Default                                                                                                                                                                                                                                                                                                 | Avg. Write Bytes: 156.21 MiB/S                                  |
| 100 partitions<br>Enhanced Kafka Configuration | environmentVariables:<br>- CONF_KAFKA_NUM_IO_THREADS=32<br>- CONF_KAFKA_NUM_NETWORK_THREADS=12<br>- CONF_KAFKA_NUM_REPLICA_FETCHERS=12<br>- CONF_KAFKA_REPLICA_SOCKET_RECEIVE_BUFFER_BYTES=1310720<br>- CONF_KAFKA_SOCKET_RECEIVE_BUFFER_BYTES=2048000<br>- CONF_KAFKA_SOCKET_SEND_BUFFER_BYTES=2048000 | Avg. Write Bytes: 175.12 MiB/S                                  |
| 5 Partitions                                   | Default                                                                                                                                                                                                                                                                                                 | Avg. Write Bytes: 149.64 MiB/S                                  |
| 5 partitions<br>acks=1                         | Default                                                                                                                                                                                                                                                                                                 | Write Bytes per Broker: 216.35 MiB/S, 175.89 MiB/S, 129.9 MiB/S |

## Kubernetes Environment Medium HWE Cinder Volume

Kubernetes Nodes:

* CPU: Intel Core Processor 6th Family 2Ghz
* Memory: 2 x DDR4-3200 16GB
* Disk: Cinder Volume Write rate 30 MB/s
* OS: Red Hat Enterprise Linux CoreOS 413.92.202308091852-0 (Plow)

### Kafka Hardware

```yaml
replicas: 3
heapSize: 2048
resources:
  requests:
    cpu: 2
    memory: 8Gi
  limits:
    cpu: 4
    memory: 8Gi
```

### Producers Setup

```yaml
replicas: 1
throughput: 50000
numRecords: 20000000
recordSize: 1024
producerProps:
  - batch.size=1000000
  - linger.ms=10
```

### Consumers Setup

```yaml
consumerGroups: ["kafka-svt-1", "kafka-svt-2", "kafka-svt-3"]
replicas: 1
fetchSize: 1048576
socketBufferSize: 2097152
timeout: 60000
messages: 20000000
```

| Scenario                                       | Kafka Configs                                                                                                                                                                                                                                                                                           | Results                                                      |
|------------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|--------------------------------------------------------------|
| 100 partitions                                 | Default                                                                                                                                                                                                                                                                                                 | Avg. Write Bytes: 18.83 MiB/S                                |
| 100 partitions<br>Enhanced Kafka Configuration | environmentVariables:<br>- CONF_KAFKA_NUM_IO_THREADS=32<br>- CONF_KAFKA_NUM_NETWORK_THREADS=12<br>- CONF_KAFKA_NUM_REPLICA_FETCHERS=12<br>- CONF_KAFKA_REPLICA_SOCKET_RECEIVE_BUFFER_BYTES=1310720<br>- CONF_KAFKA_SOCKET_RECEIVE_BUFFER_BYTES=2048000<br>- CONF_KAFKA_SOCKET_SEND_BUFFER_BYTES=2048000 | Avg. Write Bytes: 21.84 MiB/S                                |
| 5 Partitions                                   | Default                                                                                                                                                                                                                                                                                                 | Avg. Write Bytes: 18.49 MiB/S                                |
| 5 partitions<br>acks=1                         | Default                                                                                                                                                                                                                                                                                                 | Write Bytes per Broker: 26.13 MiB/S, 26.31 MiB/S, 24.5 MiB/S |

