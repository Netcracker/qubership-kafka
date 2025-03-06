This section provides information about the manual password changing procedures in the Kafka service.

# Kafka Credentials

The Kafka secret contains credentials for admin and client users to connect to Kafka.

To update Kafka credentials:

1. Navigate to **Kubernetes** > **"${KAFKA_NAMESPACE}"** > **Config and Storage** > **Secrets**.
2. Select **`${SERVICE_NAME}-secret`**.
3. Update the values for the `admin-username` and `admin-password` and/or `client-username` and `client-password` properties
   with new credentials in `BASE64` encoding. 
   For more information about these credentials,
   refer to [Kafka Server Security Properties](/docs/public/security/kafka.md#kafka-server-security-properties).
4. Don't forget to save your changes.
5. Restart Kafka brokers to apply the newly specified credentials.

where:

* `${KAFKA_NAMESPACE}` is the name of the namespace where Kafka is installed.
* `${SERVICE_NAME}` is the value of `global.name` parameter. It's `kafka` by default.

**Note:** If there are related services (e.g. Kafka Monitoring, Streaming Platform) you have to change their Kafka's credentials and
restart pods after updating Kafka secret. 

# ZooKeeper Credentials

The Kafka secret contains user credentials for SASL authentication to ZooKeeper server.

To update ZooKeeper credentials:

1. Navigate to **Kubernetes** > **"${KAFKA_NAMESPACE}"** > **Config and Storage** > **Secrets**.
2. Select **`${SERVICE_NAME}-secret`** on the `Secrets` tab.
3. Update the values for the `zookeeper-client-username` and `zookeeper-client-password` properties with new credentials 
   in `BASE64` encoding. 
   For more information about these credentials, refer to
   [ZooKeeper Authentication](/docs/public/security/kafka.md#zookeeper-authentication).
4. Don't forget to save your changes.
5. Restart Kafka to apply the newly specified credentials.

where:

* `${KAFKA_NAMESPACE}` is the name of the namespace where Kafka is installed.
* `${SERVICE_NAME}` is the value of `global.name` parameter. It's `kafka` by default.

**Note:** Make sure ZooKeeper server contains the specified credentials for the Kafka client. 
For more information about how to change credentials on the ZooKeeper side, refer to _ZooKeeper Service Installation Procedure_.
