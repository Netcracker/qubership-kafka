# How run demo

## Launch kafka using docker-compose

* Start virtual machine and go to project folder in terminal.

* Go to folder where is file docker-compose.yml and run it with command

    ```
    docker-compose up
    ```

* In new terminal check that all containers from `docker-compose.yml` have been started.

## Useful commands

Go to kafka container

  ```
  docker exec -ti <kafka_container_id> /bin/bash
  ```
    
If you need to use kafka console consumer and producer to create and get records and security 
is enabled, create file `config/security.properties` with information about security connection

  ```
  sasl.mechanism=SCRAM-SHA-512
  security.protocol=SASL_PLAINTEXT
  sasl.jaas.config=org.apache.kafka.common.security.scram.ScramLoginModule required username="admin" password="admin";
  ```

### Create topic

  ```
  bin/kafka-topics.sh --create --bootstrap-server kafka:9092 --replication-factor 2 --partitions 3 --topic testtopic --command-config config/security.properties
  ```

For example, with parameters

  ```
  bin/kafka-topics.sh --create --bootstrap-server demo_kafka_1:9092 --replication-factor 1 --partitions 1 --topic test --command-config config/security.properties
  ```
    
### Check presence of topics

  ```
  bin/kafka-topics.sh --list --bootstrap-server kafka:9092 --command-config config/security.properties
  ```
    
### Create records in topic

  ```
  bin/kafka-console-producer.sh --bootstrap-server kafka:9092 --topic testtopic --producer.config config/security.properties
  ```
    
  ```
  <first_record>
  <second_record>
  <third_record>
  ...
  ```
    
### Check presence of records in topic

  ```
  bin/kafka-console-consumer.sh --bootstrap-server kafka:9092 --topic testtopic --consumer.config config/security.properties --from-beginning
  bin/kafka-console-consumer.sh --bootstrap-server kafka:9092 --topic testtopic --group test-conusmer-group --consumer.config config/security.properties --from-beginning 
  ```

### Update replication factor of all topics to the number of brokers

  ```
  bin/kafka-partitions.sh rebalance
  ```