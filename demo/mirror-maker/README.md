# Launch kafka-mirror-maker using docker-compose

## Prepare the environment

* Start virtual machine and go to project folder in terminal.

* In this folder build image for changed component using the script

    ```
    #!/usr/bin/env bash
    
    set -e
    set -x
    
    DOCKER_FILE=docker/Dockerfile
    DOCKER_IMAGE_NAME=search/kafka-mm
    TAG=bugfix
    IMAGE_NAME="${DOCKER_IMAGE_NAME}:${TAG}"
    
    docker build \
      --file=${DOCKER_FILE} \
      -t ${IMAGE_NAME} \
      --no-cache \
      .
    
    docker inspect ${IMAGE_NAME}
    ```
  TAG variable can be changed depending on needs. IMAGE_NAME variable is used in `docker-compose.yml`.

* Go to folder where is file docker-compose.yml and run it with command

    ```
    docker-compose up
    ```

* In new terminal check that all containers from `docker-compose.yml` were started.

## Create topic in left kafka

* Go to left kafka container

    ```
    docker exec -ti <container_id> /bin/bash
    ```

* Create new topic with *test* name

    ```
    bin/kafka-topics.sh --create --bootstrap-server demo_left-kafka_1:9092 --replication-factor 1 --partitions 1 --topic test --command-config config/security.properties
    ```

## Restart kafka-mirror-maker

* Stop launched container with `kafka-mirror-maker`

    ```
    docker stop <container_id>
    ```
    
    and start it

    ```
    docker start <container_id>
    ```
    
    It's necessary because `kafka-mirror-maker` can't pick up topics dynamically.

## Create messages in left kafka

* Create file config/security.properties which contains information about security connection for consumer/producer to kafka with next content

    ```
    sasl.mechanism=SCRAM-SHA-512
    security.protocol=SASL_PLAINTEXT
    sasl.jaas.config=org.apache.kafka.common.security.scram.ScramLoginModule required username="admin" password="admin";
    ```

    Username and password should be filled with actual data for security connection.

    **Warning!!!** If password contains only digits, kafka (left and right) is failed with following error

    ```
    [ERROR][category=kafka.server.KafkaServer] Fatal error during KafkaServer startup. Prepare to shutdown
    org.apache.kafka.common.KafkaException: Exception while loading Zookeeper JAAS login context 'Client'
            at org.apache.kafka.common.security.JaasUtils.isZkSecurityEnabled(JaasUtils.java:45)
            at kafka.server.KafkaServer.initZkClient(KafkaServer.scala:358)
            at kafka.server.KafkaServer.startup(KafkaServer.scala:202)
            at kafka.server.KafkaServerStartable.startup(KafkaServerStartable.scala:38)
            at kafka.Kafka$.main(Kafka.scala:75)
            at kafka.Kafka.main(Kafka.scala)
    Caused by: java.lang.SecurityException: java.io.IOException: Configuration Error:
            Line 9: expected [option value], found [null]
            at sun.security.provider.ConfigFile$Spi.<init>(ConfigFile.java:137)
            at sun.security.provider.ConfigFile.<init>(ConfigFile.java:102)
            at sun.reflect.NativeConstructorAccessorImpl.newInstance0(Native Method)
            at sun.reflect.NativeConstructorAccessorImpl.newInstance(NativeConstructorAccessorImpl.java:62)
            at sun.reflect.DelegatingConstructorAccessorImpl.newInstance(DelegatingConstructorAccessorImpl.java:45)
            at java.lang.reflect.Constructor.newInstance(Constructor.java:423)
            at java.lang.Class.newInstance(Class.java:442)
            at javax.security.auth.login.Configuration$2.run(Configuration.java:255)
            at javax.security.auth.login.Configuration$2.run(Configuration.java:247)
            at java.security.AccessController.doPrivileged(Native Method)
            at javax.security.auth.login.Configuration.getConfiguration(Configuration.java:246)
            at org.apache.kafka.common.security.JaasUtils.isZkSecurityEnabled(JaasUtils.java:42)
            ... 5 more
    Caused by: java.io.IOException: Configuration Error:
            Line 9: expected [option value], found [null]
            at sun.security.provider.ConfigFile$Spi.ioException(ConfigFile.java:666)
            at sun.security.provider.ConfigFile$Spi.match(ConfigFile.java:579)
            at sun.security.provider.ConfigFile$Spi.parseLoginEntry(ConfigFile.java:480)
            at sun.security.provider.ConfigFile$Spi.readConfig(ConfigFile.java:427)
            at sun.security.provider.ConfigFile$Spi.init(ConfigFile.java:329)
            at sun.security.provider.ConfigFile$Spi.init(ConfigFile.java:271)
            at sun.security.provider.ConfigFile$Spi.<init>(ConfigFile.java:135)
            ... 16 more
    ```

* Send messages in the topic (if there is no topic with such name, it will be created automatically)

    ```
    bin/kafka-console-producer.sh --broker-list demo_left-kafka_1:9092 --topic test --producer.config config/security.properties
    ```

    ```
    First record
    Second record
    Third record
    ```

* Check that left kafka received all messages

    ```
    bin/kafka-console-consumer.sh --bootstrap-server demo_left-kafka_1:9092 --topic test --consumer.config config/security.properties --from-beginning
    ```

## Check state of right kafka

* Go to right kafka container

    ```
    docker exec -ti <container_id> /bin/bash
    ```
    
* Check that topic created in left kafka appeared in right kafka

    ```
    bin/kafka-topics.sh --list --bootstrap-server demo_right-kafka_1:9092 --command-config bin/security.properties
    ```
    
* Create file `config/security.properties` which contains information about security connection for consumer/producer to kafka with next content

    ```
    sasl.mechanism=SCRAM-SHA-512
    security.protocol=SASL_PLAINTEXT
    sasl.jaas.config=org.apache.kafka.common.security.scram.ScramLoginModule required username="user2" password="usertwo";
    ```
    
    Username and password should be filled with actual data for security connection.

* Check that messages in topic transferred to right kafka

    ```
    bin/kafka-console-consumer.sh --bootstrap-server demo_right-kafka_1:9092 --topic test --consumer.config config/security.properties --from-beginning
    ```