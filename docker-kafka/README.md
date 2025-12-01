# Requirements

1. [Docker](https://www.docker.io/gettingstarted/#h_installation)
2. [Docker Compose](http://docs.docker.com/compose/install/)

# Security

When using plain authentication all credentials are stored in JAAS config files 
(see [broker configuration](http://kafka.apache.org/documentation.html#security_sasl_plain_brokerconfig) 
section in kafka documentation for more info).
Example of JAAS config file for broker configuration is provided in /config folder. Kafka clients 
(producer or consumer) should use following configuration in this case:

```
KafkaClient {
    org.apache.kafka.common.security.plain.PlainLoginModule required
    username="SampleUser"
    password="SampleUserPassword";
};
```

# Monitoring

Kafka provides metrics via jmx.
More details see [Kafka Monitoring](https://kafka.apache.org/documentation/#monitoring)

For monitoring you need:

- setup link:[InfluxDB](https://github.com/influxdata/influxdb) database
- setup customized link:[Telegraf](https://github.com/influxdata/telegraf) 
  link:[Kafka Monitoring](https://github.com/Netcracker/qubership-kafka) agent 
  with InfluxDB output plugin and kafka exec input plugin
- setup link:[Grafana](https://github.com/grafana/grafana) and configure datasource (influxdb)
- import JSON template for Grafana Dashboard

Also Kafka provides JMX metrics in Prometheus format on `/metrics` REST endpoint. 
When `DISABLE_SECURITY` environment variable is set to `true`, the `/metrics` endpoint is not secured. 
When `DISABLE_SECURITY` environment variable is set to `false`, use Kafka client credentials for authentication on the `/metrics` endpoint. 

# Environment variables

The Kafka image uses several environment variables when running a Kafka broker using this image.

## `BROKER_ID`

This environment variable is recommended. Set this to the unique and persistent number for the broker. 
This must be set for every broker in a Kafka cluster, and should be set for a single standalone broker. 
The default is '1', and setting this will update the Kafka configuration.

## `REPLICATION_FACTOR`

This environment variable is recommended. Set this to the replication factor for the Kafka internal 
topics. There must be at least as many brokers as the replication factor used. 
The default is '1', and setting this will update the Kafka configuration.

## `ZOOKEEPER_CONNECT`

This environment variable is recommended, although linking to a `zookeeper` container precludes the 
need to use it. Otherwise, set this to a string described in the Kafka documentation for the 
'zookeeper.connect' property so that the Kafka broker can find the ZooKeeper service. Setting this 
will update the Kafka configuration.

## `HOST_NAME`

This environment variable is a recommended setting. Set this to the hostname that the broker will bind to. 
Defaults to the hostname of the container.

## `ADVERTISED_HOST_NAME`

This environment variable is a recommended setting. The hostname specified with this environment 
variable will be registered in ZooKeeper and given out to other workers to connect with. By default 
the value of `HOST_NAME` is used, so specify a different value if the `HOST_NAME` value will not be 
useful to or reachable by clients.

## `HEAP_OPTS`

This environment variable is recommended. Use this to set the JVM options for the Kafka broker. 
By default a value of '-Xmx640m -Xms640m' is used, meaning that each Kafka broker uses 1GB of memory. 
Using too little memory may cause performance problems, while using too much may prevent the broker 
from starting properly given the memory available on the machine. Obviously the container must be 
able to use the amount of memory defined by this environment variable.

## `LOG_LEVEL`

This environment variable is optional. Use this to set the level of detail for Kafka's application 
log written to STDOUT and STDERR. Valid values are `INFO` (default), `WARN`, `ERROR`, `DEBUG`, or `TRACE`."

## `HEALTH_CHECK_TYPE`

This environment variable is optional. Set it to `consumer_producer` to use Golang-based readiness health check, which produces and consumes data from special health-check topics.
By default Kcat-based readiness health check is used, which looks for a current broker to be present in cluster metadata list, it is more light-weight.

## Others

Environment variables that start with `CONF_KAFKA_` will be used to update the Kafka configuration file. 
Each environment variable name will be mapped to a configuration property name by:

1. removing the `CONF_KAFKA_` prefix;
2. lowercasing all characters; and
3. converting all '_' characters to '.' characters

For example, the environment variable `CONF_KAFKA_ADVERTISED_HOST_NAME` is converted to 
the `advertised.host.name` property, while `CONF_KAFKA_AUTO_CREATE_TOPICS_ENABLE` is converted to the 
`auto.create.topics.enable` property. The container will then update the Kafka configuration file to 
include the property's name and value.

The value of the environment variable may not contain a '\@' character.

# Ports

Containers created using this image will expose port 9092, which is the standard port used by Kafka.
You can  use standard Docker options to map this to a different port on the host that runs the container.

# Storing data

The Kafka broker run by this image writes data to the local file system, and the only way to keep this 
data is to use volumes that map specific directories inside the container to the local file system 
(or to OpenShift persistent volumes).

## Topic data

This image defines a data volume at `/var/opt/kafka/data`. The broker writes all persisted data as 
files within this directory, inside a subdirectory named with the value of BROKER_ID (see above). 
You must mount it appropriately when running your container to persist the data after the container 
is stopped; failing to do so will result in all data being lost when the container is stopped.

## Log files

Although this image will send Kafka broker log output to standard output so it is visible in the 
Docker logs, this image also configures Kafka to write out more detailed logs to a data volume 
at `/opt/kafka/logs`.

# Topic management

## Increasing replication factor

When upgrading kafka to a newer version that uses higher replication factor for topics you may want
to update existing topics to use higher replication factor as well. To update replication factor of
existing topics to number of available brokers in cluster following command can be used in container:
`/opt/kafka/bin/kafka-partitions.sh rebalance` or `/opt/kafka/bin/kafka-partitions.sh rebalance_topic %TOPIC_NAME%`.

In some case, you can see something like this:

```
ERROR: Assigned replicas (1) don't match the list of replicas for reassignment (1,2,3) for partition __consumer_offsets-22
...
Reassignment of partition __consumer_offsets-22 failed
...
```

If so, you could try a graceful shutdown of the controller broker and re-run the command again.

## Preparing for cluster scale down

Before performing scale down of the kafka cluster it is highle recommended to migrate topic partitions
from broker that is going to be removed from cluster to other brokers. Following command can be used in
container to perform this operation:
`/opt/kafka/bin/kafka-partitions.sh release_broker %BROKER_ID%`
where %BROKER_ID% is ID of broker that is going to be removed from cluster.
