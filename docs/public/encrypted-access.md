# Encrypted Access

## SSL Configuration

You can enable TLS-based encryption for communication with Kafka. For this you need to do the following:

### SSL Configuration using pre created Secret with manually generated Certificates

1. Create secret with certificates (or specify to generate them automatically using `generateCerts` deployment parameter):

    ```yaml
      kind: Secret
      apiVersion: v1
      metadata:
        name: ${SECRET_NAME}
        namespace: ${NAMESPACE}
      data:
        ca.crt: ${ROOT_CA_CERTIFICATE}
        tls.crt: ${CERTIFICATE}
        tls.key: ${PRIVATE_KEY}
      type: kubernetes.io/tls
    ``` 
  
    Where:
  
    * `${SECRET_NAME}` is the name of secret that contains all certificates. For example, `kafka-tls-secret`.
    * `${NAMESPACE}` is the namespace where the secret should be created. For example, `kafka-service`.
    * `${ROOT_CA_CERTIFICATE}` is the root CA in BASE64 format.
    * `${CERTIFICATE}` is the certificate in BASE64 format.
    * `${PRIVATE_KEY}` is the private key in BASE64 format.

2. Specify the following deployment parameters:

    ```yaml
    global:
      tls:
        enabled: true
        cipherSuites: []
        allowNonencryptedAccess: false
        generateCerts:
          enabled: false
          certProvider: cert-manager
          durationDays: 365
          clusterIssuerName: ""     
    kafka:
      ...
      tls:
        enabled: true
        secretName: "kafka-tls-secret"
        cipherSuites: []
        enableTwoWaySsl: false
        subjectAlternativeName:
          additionalDnsNames: []
          additionalIpAddresses: []
    ```
  
    **NOTE:** Do not use `clean` value for `DEPLOYMENT_MODE` . It removes all pre-created secrets from namespace Kafka.

### SSL Configuration using CertManager

The example of parameters to deploy Kafka services with enabled TLS and `CertManager` certificate generation:

```yaml
global:
  tls:
    enabled: true
    cipherSuites: []
    allowNonencryptedAccess: false
    generateCerts:
      enabled: true
      certProvider: cert-manager
      durationDays: 365
      clusterIssuerName: ""
    disasterRecovery:
      tls:
        enabled: true
        secretName: "kafka-drd-tls-secret"
        cipherSuites: []
        subjectAlternativeName:
          additionalDnsNames: []
          additionalIpAddresses: []
kafka:
  tls:
    enabled: true
    secretName: "kafka-tls-secret"
    cipherSuites: []
    enableTwoWaySsl: false
    subjectAlternativeName:
      additionalDnsNames: []
      additionalIpAddresses: []
backupDaemon:
  tls:
    enabled: true
    secretName: "kafka-backup-daemon-tls-secret"
    subjectAlternativeName:
      additionalDnsNames: [ ]
      additionalIpAddresses: [ ]
```

Minimal parameters to enable TLS are:

```yaml
global:
  tls:
    enabled: true
    generateCerts:
      certProvider: cert-manager
```

### SSL Configuration using parameters with manually generated Certificates

You can automatically generate TLS-based secrets using Helm by specifying certificates in deployment parameters.
For example, to generate `kafka-tls-secret` :

1. Following certificates should be generated in BASE64 format:

    ```yaml
      ca.crt: ${ROOT_CA_CERTIFICATE}
      tls.crt: ${CERTIFICATE}
      tls.key: ${PRIVATE_KEY}
    ```
  
    Where:
  
    * `${ROOT_CA_CERTIFICATE}` is the root CA in BASE64 format.
    * `${CERTIFICATE}` is the certificate in BASE64 format.
    * `${PRIVATE_KEY}` is the private key in BASE64 format.

2. Specify the certificates and other deployment parameters:

    ```yaml
    global:
      tls:
        enabled: true
        cipherSuites: []
        allowNonencryptedAccess: false
        generateCerts:
          enabled: false
          certProvider: helm     
    kafka:
      ...
      tls:
        enabled: true
        secretName: "kafka-tls-secret"
        certificates:
          crt: LS0tLS1CRUdJTiBSU0E...  
          key: LS0tLS1CRUdJTiBSU0EgUFJJV...
          ca: LS0tLS1CRUdJTiBSU0E...
        cipherSuites: []
    ```
  
    The example of parameters to deploy Kafka services with enabled TLS and Helm certificate provider:
  
    ```yaml
    global:
      tls:
        enabled: true
        cipherSuites: []
        allowNonencryptedAccess: false
        generateCerts:
          enabled: false
          certProvider: helm
        disasterRecovery:
          tls:
            enabled: true
            certificates:
              crt: LS0tLS1CRUdJTiBSU0E...  
              key: LS0tLS1CRUdJTiBSU0EgUFJJV...
              ca: LS0tLS1CRUdJTiBSU0E...
            secretName: "kafka-drd-tls-secret"      
    kafka:
      ...
      tls:
        enabled: true
        secretName: "kafka-tls-secret"
        certificates:
          crt: LS0tLS1CRUdJTiBSU0E...  
          key: LS0tLS1CRUdJTiBSU0EgUFJJV...
          ca: LS0tLS1CRUdJTiBSU0E...
        cipherSuites: []
        enableTwoWaySsl: false
        subjectAlternativeName:
          additionalDnsNames: []
          additionalIpAddresses: []
    backupDaemon:
      tls:
        enabled: true
        certificates:
          crt: LS0tLS1CRUdJTiBSU0E...  
          key: LS0tLS1CRUdJTiBSU0EgUFJJV...
          ca: LS0tLS1CRUdJTiBSU0E...
        secretName: "kafka-backup-daemon-tls-secret"
        subjectAlternativeName:
          additionalDnsNames: [ ]
          additionalIpAddresses: [ ]
    ```

## Certificate Renewal

CertManager automatically renews Certificates.
It calculates when to renew a Certificate based on the issued X.509 certificate's duration and a `renewBefore` value
which specifies how long before expiry a certificate should be renewed.
By default, the value of `renewBefore` parameter is 2/3 through the X.509 certificate's `duration`.
More info in [Cert Manager Renewal](https://cert-manager.io/docs/usage/certificate/#renewal).

After certificate renewed by CertManager the secret contains new certificate, but running applications store previous
certificate in pods.
As CertManager generates new certificates before old expired the both certificates are valid for some time (`renewBefore`).

Kafka service does not have any handlers for certificates secret changes, so you need to manually restart **all**
Kafka service pods until the time when old certificate is expired.

## Example of Client Configurations

### Java-Based Client Configurations

* SASL/PLAIN with SSL:

   ```properties
   security.protocol=SASL_SSL
   sasl.mechanism=PLAIN
   sasl.jaas.config=org.apache.kafka.common.security.plain.PlainLoginModule required username="client" password="client";
   ssl.truststore.location=/app/tls-ks/kafka.truststore.jks
   ssl.truststore.password=changeit
   ```

* SASL/SCRAM with SSL:

   ```properties
   security.protocol=SASL_SSL
   sasl.mechanism=SCRAM-SHA-512
   sasl.jaas.config=org.apache.kafka.common.security.scram.ScramLoginModule required username="client" password="client";
   ssl.truststore.location=/app/tls-ks/kafka.truststore.jks
   ssl.truststore.password=changeit
   ```

* SASL/SCRAM with two-way SSL:

   ```properties
   security.protocol=SASL_SSL
   sasl.mechanism=SCRAM-SHA-512
   sasl.jaas.config=org.apache.kafka.common.security.scram.ScramLoginModule required username="client" password="client";
   ssl.keystore.location=/app/tls-ks/kafka.keystore.jks
   ssl.keystore.password=changeit
   ssl.key.password=changeit
   ssl.truststore.location=/app/tls-ks/kafka.truststore.jks
   ssl.truststore.password=changeit
   ```

* Two-way SSL:

   ```properties
   security.protocol=SSL
   ssl.keystore.location=/app/tls-ks/kafka.keystore.jks
   ssl.keystore.password=changeit
   ssl.key.password=changeit
   ssl.truststore.location=/app/tls-ks/kafka.truststore.jks
   ssl.truststore.password=changeit
   ```

* SSL only:

   ```properties
   security.protocol=SSL
   ssl.truststore.location=/app/tls-ks/kafka.truststore.jks
   ssl.truststore.password=changeit
   ```

## Librdkafka-Based Client Configurations

* SASL/PLAIN with SSL:

   ```properties
   security.protocol=SASL_SSL
   sasl.mechanisms=PLAIN
   sasl.username=client
   sasl.password=client
   ssl.ca.location=/app/tls/ca.crt
   ```

* SASL/SCRAM with SSL:

   ```properties
   security.protocol=SASL_SSL
   sasl.mechanisms=SCRAM-SHA-512
   sasl.username=client
   sasl.password=client
   ssl.ca.location=/app/tls/ca.crt
   ```

* SASL/SCRAM with two-way SSL:

   ```properties
   security.protocol=SASL_SSL
   sasl.mechanisms=SCRAM-SHA-512
   sasl.username=client
   sasl.password=client
   ssl.ca.location=/app/tls/ca.crt
   ssl.key.location=/app/tls/tls.key
   ssl.certificate.location=/app/tls/tls.crt
   ```

* Two-way SSL:

   ```properties
   security.protocol=SSL
   ssl.ca.location=/app/tls/ca.crt
   ssl.key.location=/app/tls/tls.key
   ssl.certificate.location=/app/tls/tls.crt
   ```

* SSL only:

   ```properties
   security.protocol=SSL
   ssl.ca.location=/app/tls/ca.crt
   ```
