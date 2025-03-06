This section describes key ideas and principles of cross-datacenter replication for Kafka.
There is no need to create all configuration files manually, you just can deploy via Kafka operator.
For more information, refer to _Deployment of Kafka Service using Helm_ section in the
_[Kafka Service Installation Procedure](installation.md)_. 

To replicate data between datacenters Kafka MirrorMaker 2.0 is used.

MirrorMaker 2.0 (MM2) is a stand-alone tool for copying data between Apache Kafka clusters. MirrorMaker 2.0 is based on Kafka Connect, data 
is read from topics in the origin cluster and written to a topic with the corresponding name in the destination cluster. 

**Important:** MirrorMaker does not delete topics on remote clusters during replication between datacenters.

Suppose there are three datacenters **{dc1, dc2, dc3}** as shown in the image below. 

![Three kafka regions](/docs/public/images/kafka-mm2-3.png)

According to [KIP-382: MirrorMaker 2.0](https://cwiki.apache.org/confluence/display/KAFKA/KIP-382%3A+MirrorMaker+2.0), 
for cross-datacenter replication (XDCR), each datacenter should have a single Connect cluster that pulls records from the other datacenters
via source connectors.
So each kafka-cluster collects all messages from other regions which means here there is a unidirectional replication.

MirrorMaker configuration is placed in `kafka-mirror-maker-configuration` configmap. 

In this case, there are three instances of MM2, one for each datacenter and the following are the corresponding configmaps: 

**MM2 configuration for datacenter-1**

```yaml
...
# configure [dc1] cluster
dc1.bootstrap.servers = ip1:port1
# configure [dc2] cluster
dc2.bootstrap.servers = ip2:port2
# configure [dc3] cluster
dc3.bootstrap.servers = ip3:port3

# configure a specific source->target replication flow
dc2->dc1.enabled = false
dc3->dc1.enabled = false

topics.blacklist = dc2\.*,dc1\.*,dc3\.*
...
```

**MM2 configuration for datacenter-2**

```yaml
...
# configure [dc1] cluster
dc1.bootstrap.servers = ip1:port1
# configure [dc2] cluster
dc2.bootstrap.servers = ip2:port2
# configure [dc3] cluster
dc3.bootstrap.servers = ip3:port3

# configure a specific source->target replication flow
dc1->dc2.enabled = false
dc3->dc2.enabled = false

topics.blacklist = dc2\.*,dc1\.*,dc3\.*
...
```

**MM2 configuration in datacenter-3**

```yaml
...
# configure [dc1] cluster
dc1.bootstrap.servers = ip1:port1
# configure [dc2] cluster
dc2.bootstrap.servers = ip2:port2
# configure [dc3] cluster
dc3.bootstrap.servers = ip3:port3

# configure a specific source->target replication flow
dc1->dc3.enabled = false
dc2->dc3.enabled = false

topics.blacklist = dc2\.*,dc1\.*,dc3\.*
...
```

Where **.bootstrap.servers** specifies the kafka-cluster address. To get kafka-cluster addresses,
refer to the _Enable Access to Kafka for External Clients_ section in the
_[Kafka Service Installation Procedure](enable-external-access.md)_.

The topic that is replicated from cluster **dc_name** has the same name with prefix **dc_name.** . 
The connectors can be configured to replicate specific topics via a whitelist or regex. 
There is no need to perform replication for topics that were created as result of a replication.
For example, there is no need to copy **dc1.topic-1** from **dc-2** to **dc-3**. 
Here,`topics.blacklist` specifies topics to exclude from replication. 

Suppose, you have the following topics:

* topic1 within dc1, 
* topic2 within dc2, 
* topic3 within dc3.

After replication the topics are: 

* dc1: 
    * topic1,
    * dc2.topic2,
    * dc3.topic3, 
* dc2: 
    * topic2,
    * dc1.topic1,
    * dc3.topic3, 
* dc3: 
    * topic3, 
    * dc1.topic1,
    * dc2.topic2. 

# Helm Kafka Deployment

When you deploy Kafka using Helm, you must specify the following parameters in **mirrorMaker** section for each datacenter:

**Deployment parameters for datacenter-1**

```yaml
...
mirrorMaker:
  regionName: "dc1"
  clusters:
   - {"name": "dc1", "bootstrapServers": "ip1:port1"}
   - {"name": "dc2", "bootstrapServers": "ip2:port2"}
   - {"name": "dc3", "bootstrapServers": "ip3:port3"}
  repeatedReplication: false
...
```

**Deployment parameters for datacenter-2**

```yaml
...
mirrorMaker:
  regionName: "dc2"
  clusters:
   - {"name": "dc1", "bootstrapServers": "ip1:port1"}
   - {"name": "dc2", "bootstrapServers": "ip2:port2"}
   - {"name": "dc3", "bootstrapServers": "ip3:port3"}
  repeatedReplication: false
...
```

**Deployment parameters for datacenter-3**

```yaml
...
mirrorMaker:
  regionName: "dc3"
  clusters:
   - {"name": "dc1", "bootstrapServers": "ip1:port1"}
   - {"name": "dc2", "bootstrapServers": "ip2:port2"}
   - {"name": "dc3", "bootstrapServers": "ip3:port3"}
  repeatedReplication: false
...
```

Here `repeatedReplication` defines how to replicate topics already created via replication process. 
If `repeatedReplication = true`, all topics with names **dc1.topic-name** are copied too. 
Otherwise, topics that are created via replication are not replicated anymore.  
The default value is `false`.

## Single Message Transformation During Replication

There is an ability to transform messages during replication using custom transformations.

**Pay attention**, custom transformations are applied to all messages produced by Kafka Mirror Maker,
including system ones such as heartbeats and checkpoints, so use predicates or transformations
that allows to limit the topics to which the transformation is applied.

When you deploy Kafka using Helm, you can specify the following parameters in **mirrorMaker** section for each datacenter:

**Deployment parameters**

```yaml
...
mirrorMaker:
  transformation:
    transforms:
      - name: HeartbeatHeaderFilter
        type: com.qubership.kafka.mirror.extension.HeaderFilter
        params:
          filter.type: exclude
          headers: heartbeat,type=heartbeat
          topics: ${replication_prefix}topic1,${replication_prefix}topic2
      - name: PingHeaderFilter
        predicate: HasPingHeaderKey
        type: org.apache.kafka.connect.transforms.Filter
    predicates:
      - name: HasPingHeaderKey
        type: org.apache.kafka.connect.transforms.predicates.HasHeaderKey
        params:
          name: ping
...
```

where:

* `com.qubership.kafka.mirror.extension.HeaderFilter` - The transformation class to filter messages by headers.
  The filter provides more advanced features than `org.apache.kafka.connect.transforms.Filter` implementation in conjunction with predicates.
  - `filter.type` - The type of filter that can have `include` or `exclude` value.
  - `headers` - The comma-separated list of headers in format: `key` or `key=value`.
  - `topics` - The comma-separated list of topics and/or regexes for messages to apply the filter to.
* `org.apache.kafka.connect.transforms.Filter` - The transformation class to filter messages used in conjunction with predicates.
  For more detailed information, refer to 
  [Included transformations](https://kafka.apache.org/documentation/#org.apache.kafka.connect.transforms.Filter).
* `org.apache.kafka.connect.transforms.predicates.HasHeaderKey` - The predicate class which is true for records with at least one header 
  with the configured name. For more detailed information, refer to 
  [Predicates](hhttps://kafka.apache.org/documentation/#org.apache.kafka.connect.transforms.predicates.HasHeaderKey).

This particular transformation configuration can be used to exclude messages with `ping`, `heartbeat`, `type=heartbeat` headers from
replication.

## Kafka Mirror Maker Replication Configurator

There is an ability to declaratively change configuration of Kafka Mirror Maker. For more information, 
refer to _KMM Configurator_ section in the _[Kafka Service Installation Procedure](kmmConfigurator.md)_.
