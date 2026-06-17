This section provides information about Kafka security measures.

# Authentication

By default, Kafka connections are protected by authentication.

## Kafka Server Security Properties

To configure security in Kafka, the following parameters must be specified:

* The `ADMIN_USERNAME` parameter specifies the username of the internal Kafka administrator user.
  These credentials are used by Kafka itself. It is a mandatory parameter, and must be specified explicitly during installation.
* The `ADMIN_PASSWORD` parameter specifies the password of the internal Kafka administrator user.
  These credentials are used by Kafka itself. It is a mandatory parameter, and must be specified explicitly during installation.
* The `CLIENT_USERNAME` parameter specifies the username of the external Kafka user.
  These credentials are used by the client (producers/consumers) to establish connection with Kafka.
  It is a mandatory parameter, and must be specified explicitly during installation.
* The `CLIENT_PASSWORD` parameter specifies the password of the external Kafka user that is used by clients.
  These credentials will be used by the client (producers/consumers) to establish connection with Kafka.
  It is a mandatory parameter, and must be specified explicitly during installation.
* The `IDP_WHITELIST` parameter specifies the whitelist of trusted identity provider issuers that can be used to verify the OAuth2 access
  token signature. This parameter is used by Kafka itself.
  The default value is empty and all requests with OAuth2 authorization are declined.

## Kafka Client Configuration

To authenticate, clients must send credentials within a connection request. It requires the following settings:

* **SCRAM**

   JAAS property:

   ```properties
   sasl.jaas.config=org.apache.kafka.common.security.scram.ScramLoginModule required username="someclient" password="someclient-secret";
   ```

   The `username` and `password` properties are used by the consumer or producer to initiate connections to the Kafka server.

   SASL properties:

   ```properties
   security.protocol=SASL_PLAINTEXT
   sasl.mechanism=SCRAM-SHA-512
   ```

* **OAUTHBEARER**

   JAAS property:

   ```properties
   sasl.jaas.config=org.apache.kafka.common.security.oauthbearer.OAuthBearerLoginModule required clientId="someclient-id" clientSecret="someclient-secret" tokenEndpoint="sometoken-endpoint";
   ```

   The `clientId`, `clientSecret`, and `tokenEndpoint` properties are used by the consumer or producer to initiate connections to the Kafka server.

   SASL properties:

   ```properties
   security.protocol=SASL_PLAINTEXT
   sasl.mechanism=OAUTHBEARER
   sasl.login.callback.handler.class=com.qubership.kafka.security.oauthbearer.OAuthBearerLoginCallbackHandler
   ```

**Note**: These properties are required for any consumer and producer configurations.

For more information about Kafka Client Configuration, refer to the _Apache Kafka Official Documentation_ at 
[http://kafka.apache.org/documentation/#security_sasl_config](http://kafka.apache.org/documentation/#security_sasl_config).

## Disabling Kafka Security

The `DISABLE_SECURITY` parameter disables security, particularly the authentication, when set to `true`.
The default value is `false` and security is enabled by default.

**WARNING:** If Kafka authentication is disabled, authentication on clients must be disabled as well.
Otherwise, clients will not be able to establish connection with Kafka due to different protocols.

When the `DISABLE_SECURITY` parameter is changed for Kafka service, ensure the Kafka Monitoring 
secret contains correct client credentials for Kafka (or does not contain credentials if security is disabled) and 
reboot Kafka Monitoring. Otherwise, Kafka Monitoring is not able to connect to Kafka due to security errors.

## Dynamic addition of users to Kafka

To add a new user to a running Kafka cluster without stopping the cluster, you can use Kafka SCRAM mechanism.
Run the following command in Kafka pod:

```bash
bin/kafka-configs.sh --bootstrap-server localhost:9092 --alter --add-config 'SCRAM-SHA-256=[iterations=8192,password=${USER_PASSWORD}],SCRAM-SHA-512=[password=${USER_PASSWORD}]' --entity-type users --entity-name ${USER_NAME}
```

Where:

* `${USER_NAME}` - the name of the user to be created.
* `${USER_PASSWORD}` - the password of the user to be created.

For more information about Kafka Authentication using SASL/SCRAM, refer to the _Apache Kafka Official Documentation_ at
[https://kafka.apache.org/documentation/#security_sasl_scram](https://kafka.apache.org/documentation/#security_sasl_scram). 

# Authorization

By default, Kafka authorization is disabled.

## Enabling Kafka Authorization

The `ENABLE_AUTHORIZATION` parameter enables authorization when set to `true`. The default value is
`false` and authorization is disabled by default.
If you enable Kafka authorization, it is necessary to specify Access Control Lists (ACLs) for used clients since
all operations are denied for clients.

# ZooKeeper Authentication

To enable ZooKeeper authentication on Kafka brokers, following are the two necessary steps:

1. Set the `ZOOKEEPER_CLIENT_USERNAME` and `ZOOKEEPER_CLIENT_PASSWORD` parameters,
   which specify the username and password of the ZooKeeper client user.
2. Set the configuration property `ZOOKEEPER_SET_ACL` to `true`.
   This parameter enables Kafka to use secure ACLs when creating ZooKeeper znodes.

These parameters can be specified explicitly during installation.

## ZooKeeper Automatic ACLs

Kafka creates the following ACLs for ZooKeeper nodes if `ZOOKEEPER_SET_ACL` in `true`:

* `/controller` - `world:anyone:r, sasl:kafka:cdrwa`

* `/brokers` - `world:anyone:r, sasl:kafka:cdrwa`

* `/kafka-acl` - `sasl:kafka:cdrwa`

* `/admin` - `world:anyone:r, sasl:kafka:cdrwa`

* `/isr_change_notification` - `world:anyone:r, sasl:kafka:cdrwa`

* `/controller_epoch` - `world:anyone:r, sasl:kafka:cdrwa`

* `/consumers` - `world:anyone:cdrwa`

* `/config` - `world:anyone:cdrwa`

Where `kafka` is username of Kafka client user for ZooKeeper.

If you want to change the default permissions, you have to do it manually on ZooKeeper.
For more information about ACLs, refer to ZooKeeper Documentation.

## Migrating Clusters

If you are running a version of Kafka that does not support security or simply has security disabled,
and you want to make the cluster secure, then you need to execute the following steps to enable ZooKeeper authentication with
minimal disruption to your operations.

To enable ZooKeeper authentication:

1. Perform an installation procedure with the specified ZooKeeper client credentials and `ZOOKEEPER_SET_ACL` set to `true`.
   At the end of the upgrade, brokers are able to manipulate znodes with strict ACLs.
2. Execute the ZkSecurityMigrator tool.
   To execute the tool, use the script `bin/zookeeper-security-migration.sh` with `zookeeper.acl` set to `secure`.
   This tool traverses the corresponding sub-trees, changing the ACLs of the znodes.

It is also possible to turn off authentication in a secure cluster.

To turn off authentication:

1. Perform an installation procedure with the specified ZooKeeper client credentials, but with `ZOOKEEPER_SET_ACL` set to `false`.
   At the end of the upgrade, brokers stop creating znodes with secure ACLs, but are still able to authenticate and manipulate all znodes.
2. Execute the ZkSecurityMigrator tool.
   To execute the tool, run the script `bin/zookeeper-security-migration.sh` with `zookeeper.acl` set to `unsecure`.
   This tool traverses the corresponding sub-trees, changing the ACLs of the znodes.
3. Perform an installation procedure without ZooKeeper client credentials.

The following is an example of how to run the migration tool:

```sh
bin/zookeeper-security-migration.sh --zookeeper.acl=secure --zookeeper.connection=zookeeper:2181
```

# Audit Logs

Security events and critical operations can be logged for audit purposes.

Kafka supports logging of the following audit events:

* Authentication attempts. Logging of authentication events is possible only 
    if authentication is enabled on cluster. For more information see [Disable Kafka Security](#disabling-kafka-security).
* Authorization attempts (access to Kafka topics and other resources). Logging of authorization events is possible only 
    if authorization is enabled on cluster. For more information see [Enabling Kafka Authorization](#enabling-kafka-authorization).

To enable audit logs set the environment variable `ENABLE_AUDIT_LOGS` to `true`.

**NOTE:** Enabling the audit produces messages for all events of access to Kafka resources (successful and failed).
So it can lead to a large number of log entries.

Samples of audit logs:

```text
[2020-06-24T11:37:41,604][INFO][category=kafka.audit] CEF:1|Qubership|Kafka|2.7.1|AUTHENTICATION|Successful authentication for principal 'admin'|1|result=successful suser=admin src=172.25.0.1 authenticationType=SCRAM-SHA-512 type=audit_log_type
[2020-06-24 11:35:02,471][INFO][category=kafka.audit] CEF:1|Qubership|Kafka|2.7.1|AUTHENTICATION|[SocketServer brokerId=1] Failed authentication with /172.25.0.1 (Authentication failed during authentication due to invalid credentials with SASL mechanism SCRAM-SHA-512)|1|result=FAILED type=audit_log_type

[2020-06-24T11:37:41,606][INFO][category=kafka.audit] CEF:1|Qubership|Kafka|2.7.1|AUTHORIZATION|Principal 'admin' with client IP '172.25.0.1' is authorized to perform operation 'Read' on resource 'Topic:LITERAL:kafka-oauth-example-topic'|2|result=authorized suser=admin resource=Topic:LITERAL:kafka-oauth-example-topic src=172.25.0.1 type=audit_log_type operation=Read
[2020-06-24T11:37:43,333][INFO][category=kafka.audit] CEF:1|Qubership|Kafka|2.7.1|AUTHORIZATION|Principal '20f0d8e2-a255-493c-9512-e6a9652cbbea' with client IP '172.25.0.1' is unauthorized to perform operation 'Describe' on resource 'Topic:LITERAL:kafka-oauth-example-topic'|2|result=unauthorized suser=20f0d8e2-a255-493c-9512-e6a9652cbbea resource=Topic:LITERAL:kafka-oauth-example-topic src=172.25.0.1 type=audit_log_type operation=Describe
```

**NOTE:** The set of parameters in logs can differ for different type of events and results.
Audit logs can be found by filter `type=audit_log_type`.
