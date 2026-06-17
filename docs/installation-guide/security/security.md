This section provides information about the exposed ports, user accounts, password policies and password changing procedures in the Kafka service.

## Exposed Ports

List of ports used by Kafka and other Services. 

| Port | Service                       | Description                                                                                                                                                                                                                                                                                |
|------|-------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| 9092 | Kafka                         | Port used for client connections to kafka brokers.                                                                                                                                                                                                                                         |
| 9093 | Kafka                         | Port used for internal communication between kafka brokers.                                                                                                                                                                                                                                |
| 9094 | Kafka                         | Port for external access to the kafka broker's container.                                                                                                                                                                                                                                  |
| 9095 | Kafka                         | Port used for non-encrypted client connections to kafka brokers.                                                                                                                                                                                                                           |
| 9096 | Kafka                         | Port used for kafka controller communication if krp.cr.Spec.Kraft.Enabled is set to true.                                                                                                                                                                                                  |
| 8080 | Kafka                         | Port used for monitoring prometheus exporter.                                                                                                                                                                                                                                              |
| 8094 | Kafka Monitoring              | Port is used to receive monitoring data related to kafka via TCP protocol.                                                                                                                                                                                                                 |
| 8092 | Kafka Monitoring              | Port is used to receive monitoring data related to kafka via UDP protocol.                                                                                                                                                                                                                 |
| 8080 | Kafka Mirror Maker            | Port used by Prometheus to monitor kmm.                                                                                                                                                                                                                                                    |
| 8094 | Kafka Mirror Maker Monitoring | Port is used to receive monitoring data related to kmm via TCP protocol.                                                                                                                                                                                                                   |
| 8092 | Kafka Mirror Maker Monitoring | Port is used to receive monitoring data related to kmm via UDP protocol.                                                                                                                                                                                                                   |
| 8080 | AKHQ                          | Port used for HTTP connections to the AKHQ application. This port exposes AKHQ’s web interface for managing and monitoring Apache kafka clusters.                                                                                                                                          |
| 8443 | Backup Daemon                 | Port is used for secure communication with the Backup Daemon service when TLS is enabled. This ensures encrypted and secure data transmission.                                                                                                                                             |
| 8080 | Backup Daemon                 | Port used to manage and execute backup and restoration tasks to ensure data integrity and availability.                                                                                                                                                                                    |
| 8081 | Controller Manager            | Port is used for both liveness and readiness probes of the controller-manager container. The liveness probe checks the /healthz endpoint to ensure the container is alive, while the readiness probe checks the /readyz endpoint to determine if the container is ready to serve requests. |
| 9090 | Cruise Control                | Port used for kafka cruise control endpoints.                                                                                                                                                                                                                                              |
| 8443 | DRD                           | If TLS for Disaster Recovery is enabled the HTTPS protocol and port 8443 is used for API requests to ensure secure communication.                                                                                                                                                          |
| 8080 | DRD                           | Port used for SiteManager endpoints.                                                                                                                                                                                                                                                       |
| 8080 | Integration-tests             | Exposes the container's port to the network. It allows access to the application running in the container.                                                                                                                                                                                 |
| 8000 | LagExporter                   | Port to run the Prometheus endpoint for lagExporter service.                                                                                                                                                                                                                               |
| 8096 | Kafka                         | Port is used for Prometheus client communication if `global.monitoringType` parameter value is equal to `prometheus`.                                                                                                                                                                      |
| 8086 | Kafka                         | Port is used for Prometheus client communication if `global.monitoringType` parameter value is equal to `influxdb`.                                                                                                                                                                        |

## User Accounts

List of user accounts used for Kafka and other Services.

| Service             | OOB accounts | Deployment parameter                        | Is Break Glass account | Can be blocked | Can be deleted | Comment                                                                                              |
|---------------------|--------------|---------------------------------------------|------------------------|----------------|----------------|------------------------------------------------------------------------------------------------------|
| Kafka               | admin        | global.secrets.kafka.adminUsername          | yes                    | no             | no             | The default admin user. There is no default value, the name must be specified during deploy.         |
| Kafka               | client       | global.secrets.kafka.clientUsername         | no                     | yes            | yes            | The default client user. There is no default value, the name must be specified during deploy.        |
| Cruise Control      | admin        | global.secrets.cruiseControl.adminUsername  | yes                    | no             | no             | The cruise control admin user. There is no default value, the name must be specified during deploy.  |
| Cruise Control      | viewer       | global.secrets.cruiseControl.viewerUsername | no                     | yes            | yes            | The cruise control viewer user. There is no default value, the name must be specified during deploy. |
| AKHQ                | client       | global.secrets.akhq.defaultUsername         | no                     | yes            | yes            | The default akhq user. There is no default value, the name must be specified during deploy.          |
| AKHQ                | client       | global.secrets.akhq.schemaRegistryUsername  | no                     | yes            | yes            | The akhq schema registry user. There is no default value, the name must be specified during deploy.  |
| Kafka Backup Daemon | client       | global.secrets.backupDaemon.username        | no                     | yes            | yes            | Backup Daemon user. There is no default value, the name must be specified during deploy.             |

## Disabling User Accounts

Kafka does not support disabling user accounts.

## Password Policies 

* Passwords must be at least 8 characters long. This ensures a basic level of complexity and security.
* The passwords can contain only the following symbols:
    * Alphabets: a-zA-Z
    * Numerals: 0-9
    * Punctuation marks: ., ;, !, ?
    * Mathematical symbols: -, +, *, /, %
    * Brackets: (, ), {, }, <, >
    * Additional symbols: _, |, &, @, $, ^, #, ~

**Note**: To ensure that passwords are sufficiently complex, it is recommended to include:

* A minimum length of 8 characters
* At least one uppercase letter (A-Z)
* At least one lowercase letter (a-z)
* At least one numeral (0-9)
* At least one special character from the allowed symbols list

## Changing password guide

Kafka Service supports the automatic password change procedure.
Any credential in the `global.secrets.kafka` section can be changed and run with upgrade procedure.
Operator performs necessary logic to apply new credentials to Kafka, AKHQ and Backup Daemon pods.

**Note:** It requires full cluster restart.

The manual password changing procedures for Kafka Service is described in respective guide:

* [Password changing guide](/docs/public/password-changing.md)

# Logging

Security events and critical operations should be logged for audit purposes.
You can find more details about available audit logging in [Audit Guide](/docs/public/audit.md).
