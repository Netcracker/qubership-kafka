# qubership-docker-kafka-mirror-maker

Kafka's mirroring feature makes it possible to maintain a replica of an existing Kafka cluster. The
Mirror Maker 2.0 (MM2) is based on the Kafka Connect framework and has the ability to keep the topic
properties in sync across clusters and improve performance significantly by reducing rebalances to a
minimum. The tool creates source connector, that reads from source Kafka cluster and writes to target
Kafka cluster, for every necessary replication flow.
More information about MM2 can be found in [KIP-382](https://cwiki.apache.org/confluence/display/KAFKA/KIP-382).