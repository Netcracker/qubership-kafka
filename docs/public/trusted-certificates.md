It is possible to configure a trusted keystore with specified certificates for Kafka.

To configure this, you have to create an OpenShift secret from a file using the following command:

```bash
oc create secret generic ${SERVICE_NAME}-trusted-certs --from-file=./certs/
```

Where:

* `certs` is the directory with public certificates for importing.
* `${SERVICE_NAME}` is the name of the Kafka service in the OpenShift project.

**Note**: Kafka creates a default empty secret when deploying. Therefore, it is necessary to delete this secret before creating the new one.

These certificates are imported to the Kafka trusted keystore with aliases as certificate file names. 
This keystore also contains all certificates from the Java keystore of the Docker image's **${JAVA_HOME}**.

**Pay attention** if you want to import CA-signed certificate, you also need to import public root-CA certificate.

After creating secret you need to restart the Kafka service.

# Certificates Format

Kafka supports importing certificates in the PEM format. For example:

```text
-----BEGIN CERTIFICATE-----
MIIFETCCA/mgAwIBAgIKLgPjLwABAAAAPjANBgkqhkiG9w0BAQUFADBKMRUwEwYK
CZImiZPyLGQBGRYFbG9jYWwxFjAUBgoJkiaJk/IsZAEZFgZ0ZXN0YWQxGTAXBgNV
BAMTEHRlc3RhZC1URVNUREMtQ0EwHhcNMTcwMzMwMDcxMzAyWhcNMTkwMzMwMDcx
MzAyWjAeMRwwGgYDVQQDExN0ZXN0ZGMudGVzdGFkLmxvY2FsMIIBIjANBgkqhkiG
9w0BAQEFAAOCAQ8AMIIBCgKCAQEAupGqI6n5sreiTxQKsPL6omtTVzGei0JJa7QY
UnY8398Yh/DFj/xRz2RFdIN/0xUeQO0LiEgR+ZwGzxKpQU7818dqWKNwzdy1sR4/
izL9UHyKsLThnAiURTqeDEHjO707AgoB00I3izYqGFR39/U/x4sCChqC1O1IdZAY
3p6IlyqJuuuWKbdN5yq6Qh9DBmUVhX5iVTj5VQd/PiFRalG5PR+jHsa0MCasPcUs
HXlP0YX5gfrsOJ6Guif+VIrHt7Acri/CbUyPYEKXt1kSwOe8PYqvAf9pRNqf63QW
tLHqtMHNee98g96GDzahiw7Zva4mEES7fkCYYpnSAeRWHWJ6TwIDAQABo4ICIzCC
Ah8wIQYJKwYBBAGCNxQCBBQeEgBXAGUAYgBTAGUAcgB2AGUAcjATBgNVHSUEDDAK
BggrBgEFBQcDATAOBgNVHQ8BAf8EBAMCBaAwHQYDVR0OBBYEFODgJ+l89al+4lQo
Tjxwe+ZMClBQMB8GA1UdIwQYMBaAFAfdjkzLR2cMUn9JuAQRL1mys0NRMIHOBgNV
HR8EgcYwgcMwgcCggb2ggbqGgbdsZGFwOi8vL0NOPXRlc3RhZC1URVNUREMtQ0Es
Q049VEVTVERDLENOPUNEUCxDTj1QdWJsaWMlMjBLZXklMjBTZXJ2aWNlcyxDTj1T
ZXJ2aWNlcyxDTj1Db25maWd1cmF0aW9uLERDPXRlc3RhZCxEQz1sb2NhbD9jZXJ0
aWZpY2F0ZVJldm9jYXRpb25MaXN0P2Jhc2U/b2JqZWN0Q2xhc3M9Y1JMRGlzdHJp
YnV0aW9uUG9pbnQwgcMGCCsGAQUFBwEBBIG2MIGzMIGwBggrBgEFBQcwAoaBo2xk
YXA6Ly8vQ049dGVzdGFkLVRFU1REQy1DQSxDTj1BSUEsQ049UHVibGljJTIwS2V5
JTIwU2VydmljZXMsQ049U2VydmljZXMsQ049Q29uZmlndXJhdGlvbixEQz10ZXN0
YWQsREM9bG9jYWw/Y0FDZXJ0aWZpY2F0ZT9iYXNlP29iamVjdENsYXNzPWNlcnRp
ZmljYXRpb25BdXRob3JpdHkwDQYJKoZIhvcNAQEFBQADggEBABCyuuRDIeTGCygV
iV/E1AVls4X6hn6IadyE6e1TnyZR9dF0wjW55r5tNsUjgpTI+fIegowggdbUPUl3
qLrPRTH5dRkwCotohQANAuf8Djqxx6HR1DkVuwSy2EeLOb9OS6BEgcfJKTgbmXEx
P6PYntwx/uXdtP2b9Vw2gHmCFuQ/PGZh2dY7ws1p0EYg5kRaa5XQ5zLiauhdHJ5m
tcLKrLUs8fivSWfh+xMWsQhwUoz7fR1elqNAU2XWfLXKB8vfFLEXfJsNOJGYuteP
JHxQuThMQSQGayNaVDtSeE+WCzZl8W2eBZe5ar6sImnrVezkb6RAMcaEFKIkaSsf
ecgdGMc=
-----END CERTIFICATE-----
```

## Kafka Console Client Usage

To enable Kafka console client to use your trusted certificates, you need to specify the following java properties with trusted keystore information:

```text
-Djavax.net.ssl.trustStore=/opt/kafka/config/cacerts -Djavax.net.ssl.trustStorePassword=changeit
```

You can do it using the local environment variable `KAFKA_OPTS`, for example:

```sh
export KAFKA_OPTS="-Djavax.net.ssl.trustStore=/opt/kafka/config/cacerts -Djavax.net.ssl.trustStorePassword=changeit" && ./bin/kafka-console-consumer.sh
```

or with environment variable `KAFKA_HEAP_OPTS` in Kafka Deployment Config, for example:

```sh
oc set env dc kafka-1 KAFKA_HEAP_OPTS="-Xms256m -Xmx256m -Djavax.net.ssl.trustStore=/opt/kafka/config/cacerts -Djavax.net.ssl.trustStorePassword=changeit"
```
