It is possible to configure a keystore with specified public certificates for OAuth token signature validation.

To configure this, you need to create an OpenShift secret from a file using the following command:

```bash
oc create secret generic ${SERVICE_NAME}-public-certs --from-file=./certs/
```

Where:

* `certs` is the directory with public certificates for importing. The name of the certificate must match the key ID.
* `${SERVICE_NAME}` is the name of the Kafka service in the OpenShift project.

**Note**: Kafka creates a default empty secret during deployment.
Therefore, it is necessary to delete this secret before creating the new one or before updating. 

These certificates are imported to the Kafka OAuth keystore with aliases as file names without extension. 
These aliases are used as `kid` (JWS key ID) during token validation. Therefore, it is important to name files by keys IDs.

For example, if you have the following public key in Keycloak:

![Keycloak Keys](/docs/public/images/keycloak.JPG)

You need to create file `Ot0ZnJHbnXAnjj5BTLmL9b270EXjj_Lw5Og5uJflzrc.crt` with the contents of `Certificate`.

After creating or updating the secret, you need to restart Kafka brokers. You can restart them node by node.

# Certificates Format

Kafka supports importing certificates in the PEM format. For example:

```text
-----BEGIN CERTIFICATE-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEApiqLTt9OznkarUPIdw8h
VSP2yYjKZPTZ5Jb+6J0yL4lArFVZ/NcssIWJfVqFJr6Dieoudju3sb+XV7bDrkIk
26Au+vGvkpOs4SsQAyOoBZqj0I3IqpDrWciRH6my5rhbebQj+CxDUSxWw46NWZ8p
tLGQ9bZaSHA9TRRQ3pr2IihCsA7a0pFg8MpL5n3EcV11/YDWRhM6P9Bafd/5/4ll
SlANIZBEVEjb/q7sIOHqRgtoJUymQMMhNJCt0Un8OHdCpRTQfS/vtFcOlorHF8Cg
oJ6ar5q0T7A3ItBwhREPQMo0sbxaTnGVkUCizG4Q/chi/WeIZpmXpMMf4l//ZRie
3wIDAQAB
-----END CERTIFICATE-----
```

**Note**: The sections `-----BEGIN CERTIFICATE-----` and `-----END CERTIFICATE-----` are required for certificate. 
If you have certificate's value without these sections, you need to add them in the file before creating the secret.
You can do it using tool [Format an X.509 certificate](https://www.samltool.com/format_x509cert.php).

## Kafka Keystore Usage

To enable Kafka to use keystore with certificates from secret to verify OAuth tokens,
you need to set the value of `JWK_SOURCE_TYPE` environment variable to `keystore` for each Kafka broker.
