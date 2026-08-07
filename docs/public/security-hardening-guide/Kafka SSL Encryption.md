# Encryption with SSL

By default, Apache Kafka communicates in `PLAINTEXT`, which means that all data is sent in the clear. 
To encrypt communication, you can configure all the Confluent Platform components in your deployment to use SSL encryption.

**Note:** Secure Sockets Layer (SSL) is the predecessor of Transport Layer Security (TLS), and SSL has been deprecated since June 2015. 
However, for historical reasons, Kafka (like Java) uses the term SSL instead of TLS in configuration and code, which can be a bit confusing. 
This document uses only the term SSL.

Note that enabling SSL may have a performance impact due to encryption overhead.

SSL uses private-key/certificates pairs which are used during the SSL handshake process.

SSL can be configured for encryption or authentication. 
You can configure just SSL encryption (by default, SSL encryption includes certificate authentication of the server) 
and independently choose a separate mechanism for client authentication (for example, SSL, SASL). 
Technically speaking, SSL encryption already enables 1-way authentication in which the client authenticates the server 
certificate. So `SSL authentication`, is really referring to 2-way authentication, where the broker also authenticates 
the client certificate.

## 1-way authentication (SSL Encryption)

### Server settings

You should create necessary Keystores and Trust Stores for Kafka server.
To enable SSL for `INTERNAL` clients on working Kafka cluster add the following environment variables to `DeploymentConfig`:

```
CONF_KAFKA_SSL_TRUSTSTORE_LOCATION=/var/private/ssl/kafka.server.truststore.jks
CONF_KAFKA_SSL_TRUSTSTORE_PASSWORD=123456
CONF_KAFKA_SSL_KEYSTORE_LOCATION=/var/private/ssl/kafka.server.keystore.jks
CONF_KAFKA_SSL_KEYSTORE_PASSWORD=123456
CONF_KAFKA_SSL_KEY_PASSWORD=123456
LISTENER_SECURITY_PROTOCOL_MAP=INTERNAL:SASL_SSL,INTER_BROKER:SASL_PLAINTEXT
CONF_KAFKA_SSL_ENDPOINT_IDENTIFICATION_ALGORITHM=
```

Note that the folder with Keystores and Trust Stores (`/var/private/ssl` in this example) should be placed on PV. 

In this example Kafka has two interfaces:

- `INTERNAL` (used by clients in OpenShift environment): port `9092`, Encrypted with SSL, SASL Authentication enabled.
- `INTER_BROKER` (used by Kafka cluster to communicate between brokers): port `9093`, Plaintext (without SSL), SASL Authentication enabled.

If you want to enable SSL for external clients (outside Openshift), specify:

```
LISTENER_SECURITY_PROTOCOL_MAP=EXTERNAL:SASL_SSL,INTERNAL:SASL_PLAINTEXT,INTER_BROKER:SASL_PLAINTEXT
```

### Client settings

You should create necessary Trust Stores for Kafka clients and create `ssl_client.conf` file with the next content:

```
ssl.truststore.location=/var/private/ssl/kafka.client.truststore.jks
ssl.truststore.password=123456
ssl.endpoint.identification.algorithm=
security.protocol=SASL_SSL
sasl.mechanism=SCRAM-SHA-512
sasl.jaas.config=org.apache.kafka.common.security.scram.ScramLoginModule required username="client" password="client";
```

**Note:** "ssl.endpoint.identification.algorithm=" is an optional parameter, which is necessary 
if your server certificate does not contain Subject Alternative Name for your Kafka cluster address. 
This property allows to skip this check.

### Check connection

You can check connection using the following command:

```
./bin/kafka-console-producer.sh --broker-list kafka:9092 --topic TOPIC_NAME --producer.config ./ssl_client.conf
./bin/kafka-console-consumer.sh --bootstrap-server kafka:9092 --topic TOPIC_NAME --consumer.config ./ssl_client.conf --from-beginning
```

### Official documentation

For more information and examples about Kafka encryption with SSL see Confluent official [Encryption with SSL](https://docs.confluent.io/platform/current/security/protect-data/encrypt-tls.html).

## 2-way authentication (Encryption and Authentication with SSL)

**Note:** With the Certificate Authority (CA) method, Kafka does not conveniently support blocking authentication 
for individual brokers or clients that were previously trusted using this mechanism. 
Kafka does not perform certificate renegotiations or revocations. In cases where a certificate is compromised 
you can't revoke a certificate. So you would have to rely on authorization to block access.

Therefore it is recommended to consider SSL channel encryption (1-way authentication) + SASL or OAuth client authentication.

### Server settings

You should create necessary Keystores and Trust Stores for Kafka server.
To enable SSL for `INTERNAL` clients on working Kafka cluster add the following environment variables to `DeploymentConfig`:

```
CONF_KAFKA_SSL_TRUSTSTORE_LOCATION=/var/private/ssl/kafka.server.truststore.jks
CONF_KAFKA_SSL_TRUSTSTORE_PASSWORD=123456
CONF_KAFKA_SSL_KEYSTORE_LOCATION=/var/private/ssl/kafka.server.keystore.jks
CONF_KAFKA_SSL_KEYSTORE_PASSWORD=123456
CONF_KAFKA_SSL_KEY_PASSWORD=123456
CONF_KAFKA_SSL_CLIENT_AUTH=required
LISTENER_SECURITY_PROTOCOL_MAP=INTERNAL:SSL,INTER_BROKER:SASL_PLAINTEXT
CONF_KAFKA_SSL_ENDPOINT_IDENTIFICATION_ALGORITHM=
```

Note that the folder with Keystores and Trust Stores (`/var/private/ssl` in this example) should be placed on PV.

In this example Kafka has two interfaces:

- `INTERNAL` (used by clients in OpenShift environment): port `9092`, Encrypted with SSL, SSL Authentication enabled.
- `INTER_BROKER` (used by Kafka cluster to communicate between brokers): port `9093`, Plaintext (without SSL), SASL Authentication enabled.

Pay attention to the environment variable `CONF_KAFKA_SSL_CLIENT_AUTH=required`, 
it enables brokers to authenticate clients' certificates.
 
If you want to enable SSL for external clients (outside Openshift), specify:

```
LISTENER_SECURITY_PROTOCOL_MAP=EXTERNAL:SSL,INTERNAL:SASL_PLAINTEXT,INTER_BROKER:SASL_PLAINTEXT
```

### Client settings

You should create necessary Keystores and Trust Stores for Kafka clients 
and create `ssl_client.conf` file with the next content:

```
security.protocol=SSL
ssl.truststore.location=/var/private/ssl/kafka.client.truststore.jks
ssl.truststore.password=123456
ssl.keystore.location=/var/private/ssl/kafka.client.keystore.jks
ssl.keystore.password=123456
ssl.key.password=123456
ssl.endpoint.identification.algorithm=
```

**Note:** "ssl.endpoint.identification.algorithm=" is an optional parameter, which is necessary if your server certificate does not contain Subject Alternative Name 
for your Kafka cluster address. This property allows to skip this check.

### Check connection

You can check connection using the following command:

```
./bin/kafka-console-producer.sh --broker-list kafka:9092 --topic TOPIC_NAME --producer.config ./ssl_client.conf
./bin/kafka-console-consumer.sh --bootstrap-server kafka:9092 --topic TOPIC_NAME --consumer.config ./ssl_client.conf --from-beginning
```

### Official documentation

For more information and examples about Kafka encryption and authentication with SSL see Confluent official [Encryption and Authentication with SSL](https://docs.confluent.io/platform/current/security/authentication/mutual-tls/overview.html).
