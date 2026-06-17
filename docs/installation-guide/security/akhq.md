AKHQ is an open source GUI for Apache Kafka to manage topics, topics data, consumer groups, Kafka Connect and more.
For more information and full list of features, refer to the [Official AKHQ Documentation](https://github.com/tchiotludo/akhq/tree/0.17.0).

# AKHQ Security Configuration

Security is enabled in AKHQ by default and cannot be turned off.

## Simple Security Configuration
 
In the simplest case, you can use a single basic user to access the AKHQ UI. 
Specify the values for AKHQ deployment parameters `DEFAULT_USER` and `AKHQ_DEFAULT_PASSWORD`. Use these credentials to access the AKHQ UI.
No additional configuration is required.

You can use this configuration on development environments. 

## Enhanced Security Configuration

On production environment, you need to be careful when you configure the security access to AKHQ.

To enhance security, you can configure separate basic users or LDAP users:

* For basic authentication users, the login and password in sha256 are **stored in AKHQ**. These users can bound with custom or default
  groups configured in AKHQ.
* For LDAP authentication users, the login and password are **stored on LDAP server**. These users can bound with custom or default groups
  configured in AKHQ. Also LDAP groups can be bound with AKHQ groups.
   
In this case, the `DEFAULT_USER` and `AKHQ_DEFAULT_PASSWORD` parameters can be left empty.

For production environment, LDAP is recommended.

To configure security for AKHQ, you need to use default pre-created security groups or create custom groups, describe basic or LDAP users
and match them with the desired security groups. For more information about security group and roles see the sections below.

### Security Groups and Roles

To specify user grants, security groups with roles are used.
The following three default groups are available:

* `admin` - A group with all rights
* `reader` - A group with only read access on all AKHQ
* `no-roles` - A group without any roles, that force user to login

You can also create custom groups.

To provide groups rights for read/write/insert/update/delete operations with topics, topics configs, ACLs and so on, Roles are used. 
To filter topics available for current group, you can use `topics-filter-regexp`. 
By default, every group has access to all topics.

>> **Please note**
> that since version 0.25.0 the group management has changed, the previous configuration format is no longer suitable.

To configure custom security groups:

1. Prepare YAML configuration in the following format:<br>

   **Old Format (kafka-service before release-2024.4-1.11.0):**

      ```yaml
      groups:
        group-1: # unique key
          name: custom-group-name # custom group name
          roles: # full list of possible values is placed below
            - topic/read
            - topic/insert
            - topic/delete
            - topic/config/update
            - node/read
            - node/config/update
            - topic/data/read
            - topic/data/insert
            - topic/data/delete
            - group/read
            - group/delete
            - group/offsets/update
            - acls/read
            - connect/read
            - connect/insert
            - connect/update
            - connect/delete
            - connect/state/update
          attributes: # optional parameter to filter topics available for current group, by default all topics are available
            topics-filter-regexp: "regexp"
      ```
   
   **New Format (kafka-service: release-2024.4-1.11.0)** <br>
   The main changes are related to the declaration of `roles` with `resources` and `actions`,
   and the declaration of user `groups` based on the composition of `roles`.
   Please bear in mind that specifying a default group will impact all custom groups,
   as all custom groups will only be an enhancement of the default or just append to the default.
   
      ```yaml
       roles:
         node-read:
           - resources: [ "NODE" ]
             actions: [ "READ", "READ_CONFIG" ]
         node-admin:
           - resources: [ "NODE" ]
             actions: [ "READ", "READ_CONFIG", "ALTER_CONFIG" ]
         topic-read:
           - resources: [ "TOPIC", "TOPIC_DATA" ]
             actions: [ "READ" ]
           - resources: [ "TOPIC" ]
             actions: [ "READ_CONFIG" ]
         topic-admin:
           - resources: [ "TOPIC", "TOPIC_DATA" ]
             actions: [ "READ", "CREATE", "DELETE" ]
           - resources: [ "TOPIC" ]
             actions: [ "UPDATE", "READ_CONFIG", "ALTER_CONFIG" ]
         connect-rw:
           - resources: [ "CONNECTOR" ]
             actions: [ "READ", "CREATE", "UPDATE_STATE" ]
         connect-admin:
           - resources: [ "CONNECTOR" ]
             actions: [ "READ", "CREATE", "UPDATE_STATE", "DELETE" ]
         registry-read:
           - resources: [ "SCHEMA" ]
             actions: [ "READ" ]
         registry-admin:
           - resources: [ "SCHEMA" ]
             actions: [ "READ", "CREATE", "UPDATE", "DELETE", "DELETE_VERSION" ]
         group-read:
           - resources: [ "CONSUMER_GROUP" ]
             actions: [ "READ" ]
         connect-cluster-read:
           - resources: [ "CONNECT_CLUSTER" ]
             actions: [ "READ" ]
         ksqldb-admin:
           - resources: [ "KSQLDB" ]
             actions: [ "READ", "EXECUTE" ]
   
      default-group: no-roles # Default groups for all the user even unlogged user
      # Groups definition
      groups:
         test-admin:
           - role: node-admin
           - role: topic-admin
             patterns: [".*test.*"]
           - role: connect-admin
           - role: registry-admin
           - role: group-read
           - role: connect-cluster-read
           - role: ksqldb-admin
         hello-topic-reader:
           - role: topic-read
             patterns: ["hello-topic"]
           - role: group-read
      ```
   
   The following example configures two custom AKHQ groups:
   
   * `test-admin` group that has full access to topics matching the `.*test.*` regexp
   * `hello-topic-reader` group that has read access to the `hello-topic`
   
   **Note**: Be careful with indentation, the YAML format is very sensitive to them.
   
2. Encode prepared YAML configuration to Base64 format and copy the encoded value. For example, you can use online resource
   [base64encode.org](https://www.base64encode.org/) for enconding.
3. Go to `akhq-security-configuration` secret in OpenShift/Kubernetes and paste the encoded value derived from the previous step in the
   `security_groups_config` field.
4. Reload AKHQ pod, scale it down and then scale up, for the change to take effect. Or reload it when all the security configuration
   (security groups, users and LDAP) is completed.

### Basic Users

For basic authentication users, the login and password in sha256 are **stored in AKHQ**. These users can bound with custom or default groups
configured in AKHQ.

If it is required to have several basic users to access AKHQ UI, do the following:

1. Prepare YAML configuration in the following format:

   ```yaml
   basic-auth:  
     - username: user-name
       password: password-in-sha256
       groups:
       - group-name
   ```

   **Note**: The password should be specified in sha256, you can convert it using the following command: 
   ```echo -n "password" | sha256sum```.

   The following example configures four basic users:
   * `user-1` with password `123` (converted to sha256) which matches custom `test-admin` group
   * `user-2` with password `123` (converted to sha256) which matches default `admin` group
   * `user-3` with password `123` (converted to sha256) which matches custom `hello-topic-reader` group
   * `user-4` with password `123` (converted to sha256) which matches default `reader` group

   ```yaml
   basic-auth:  
     - username: user-1
       password: a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3
       groups:
       - test-admin
     - username: user-2
       password: a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3
       groups:
       - admin
     - username: user-3
       password: a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3
       groups:
       - hello-topic-reader
     - username: user-4
       password: a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3
       groups:
       - reader
   ```

   **Note**: Be careful with indentation, the YAML format is very sensitive to them.   

2. Encode prepared YAML configuration to Base64 format and copy the encoded value. 
   For example, you can use online resource [base64encode.org](https://www.base64encode.org/) for enconding.
3. Go to `akhq-security-configuration` secret in OpenShift/Kubernetes and paste the encoded value derived from the previous step in
   the `basic_auth_users_config` field.
4. Reload AKHQ pod, scale it down and then scale up, for the change to take effect. 
   Or reload it when all the security configuration (security groups, users and LDAP) is completed.

### LDAP Server Configuration

To configure LDAP server connection:

1. Prepare YAML configuration in the following format:

   ```yaml
   context:
     server: 'ldap://ldap-server-host:ldap-server-port'
     managerDn: 'manager DN'
     managerPassword: 'manager password'
   search:
     # search configuration
   groups: # optional
     enabled: true
     # groups configuration
   ```

   AKHQ is based on Micronaut, and Micronaut supports authentication with LDAP out of the box. For more information about LDAP parameters 
   in Micronaut, refer to 
   [Micronaut LDAP Authentication Official Documentation](https://micronaut-projects.github.io/micronaut-security/latest/guide/#ldap).

   LDAP server configuration example for `testad.example.com`:

   ```yaml
   context:
     server: 'ldap://testad.example.com:389'
     managerDn: 'testDN'
     managerPassword: 'test'
   search:
     base: "DC=testad,DC=local"
     filter: "(&(objectClass=user)(cn={0}))"
   ```

   **Note**: Be careful with indentation, the YAML format is very sensitive to them.

2. Encode prepared YAML configuration to Base64 format and copy the encoded value.
   For example, you can use the online resource [base64encode.org](https://www.base64encode.org/) for encoding.
3. Go to `akhq-security-configuration` secret in OpenShift/Kubernetes and paste the encoded value derived from the previous step in
   the `ldap_server_config` field.
4. Reload AKHQ pod, scale it down and then scale up, for the change to take effect.
   Or reload it when all the security configuration (security groups, users and LDAP) is completed.

### LDAPS Server Configuration

LDAPS is a secured LDAP (LDAP over TLS). It is more suitable for production environment.

To configure LDAPS server connection:

1. Prepare YAML configuration in the following format:

   ```yaml
   context:
     server: 'ldaps://ldap-server-host:ldap-server-port'
     managerDn: 'manager DN'
     managerPassword: 'manager password'
   search:
     # search configuration
   groups: # optional
     enabled: true
     # groups configuration
   ```

   AKHQ is based on Micronaut, and Micronaut supports authentication with LDAP out of the box. 
   For more information about LDAP parameters in Micronaut, refer to 
   [Micronaut LDAP Authentication Official Documentation](https://micronaut-projects.github.io/micronaut-security/latest/guide/#ldap).

   LDAPS server configuration example for `testad.example.com`. The server address is specified as `dc.testad.local`,
   which is per the address specified in the certificate:

   ```yaml
   context:
     server: 'ldaps://dc.testad.local:636'
     managerDn: 'testDN'
     managerPassword: 'test'
   search:
     base: "DC=testad,DC=local"
     filter: "(&(objectClass=user)(cn={0}))"
   ``` 

   **Note**: Be careful with indentation, the YAML format is very sensitive to them.

2. Encode prepared YAML configuration to Base64 format and copy the encoded value. For example, you can use the online resource 
   [base64encode.org](https://www.base64encode.org/) for encoding.
3. Go to `akhq-security-configuration` secret in OpenShift/Kubernetes and paste the encoded value derived from the previous step in
   the `ldap_server_config` field.
4. Load LDAPS server certificate to AKHQ Trust Store.
   For more information, see [LDAPS Trusted Certificates Importing](#ldaps-trusted-certificates-importing).
5. Reload AKHQ pod, scale it down and then scale up, for the change to take effect.
   Or reload it when all the security configuration (security groups, users and LDAP) is completed.

### LDAPS Trusted Certificates Importing

It is possible to configure a trusted Keystore with specified LDAPS certificates for AKHQ.

To configure this, you need to create an OpenShift secret from a file using the following command:

```sh
oc create secret generic akhq-trusted-certs --from-file=./certs/
```

Where:

* `certs` is the directory with public LDAPS certificates for importing.

**Note**: AKHQ creates a default empty secret when deploying. Therefore, it is necessary to delete this secret before creating the new one. 

These certificates are imported to the AKHQ trusted Keystore with aliases as certificate file names. 
This Keystore also contains all certificates from the Java Keystore of the Docker image's **${JAVA_HOME}**.

**Pay attention** if you want to import CA-signed certificate, you also need to import public root-CA certificate.

### LDAP Users

For LDAP authentication users, the login and password are **stored on LDAP server**, 
it is more secure, and therefore LDAP/LDAPS is recommended for production environment.

To configure LDAP users and groups:

1. Prepare YAML configuration in the following format:

   ```yaml
   ldap:
     default-group: no-roles
     groups: # allows to match LDAP groups with AKHQ groups
       - name: group-ldap-1
         groups: # Akhq groups list
           - akhq-group
     users: # allows to match LDAP users with AKHQ groups
       - username: ldap-user-id
         groups: # Akhq groups list
           - akhq-group
   ```

   The following example configures two LDAP users:
   * `user-1` which matches custom AKHQ `test-reader` group
   * `user-2` which matches default AKHQ `admin` group

   ```yaml
   ldap:  
     users:
       - username: user-1
         groups:
           - topic-reader
       - username: user-2
         groups:
           - admin
   ```

   **Note**: Be careful with indentation, the YAML format is very sensitive to them.   

2. Encode prepared YAML configuration to Base64 format and copy the encoded value.
   For example, you can use online resource [base64encode.org](https://www.base64encode.org/) for encoding.
3. Go to `akhq-security-configuration` secret in OpenShift/Kubernetes and paste the encoded value derived from
   the previous step in the `ldap_users_config` field.
4. Reload AKHQ pod, scale it down and then scale up, for the change to take effect.
   Or reload it when all the security configuration (security groups, users and LDAP) is completed.

### LDAP Debug

LDAP configuration can be tricky.
To debug LDAP connection, execute the following command:

```bash
curl -i -X POST -H "Content-Type: application/json" \
       -d '{ "configuredLevel": "TRACE" }' \
       http://<AKHQ URL>/loggers/io.micronaut.configuration.security
```

It increases the log level for the current session on the Micronaut part that handles the LDAP authentication.
